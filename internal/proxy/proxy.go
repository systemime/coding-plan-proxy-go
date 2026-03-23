// Package proxy 提供 API 代理转发功能
package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"coding-plan-mask/internal/config"
	"coding-plan-mask/internal/ratelimit"
	"coding-plan-mask/internal/storage"

	"go.uber.org/zap"
)

// Proxy API 代理
type Proxy struct {
	cfg       *config.Config
	rateLimit *ratelimit.GlobalLimiter
	client    *http.Client
	logger    *zap.Logger
	storage   *storage.Storage
	output    io.Writer
}

// New 创建新的代理实例
func New(cfg *config.Config, logger *zap.Logger, store *storage.Storage) *Proxy {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
	}

	rateLimiter := ratelimit.NewGlobalLimiter(cfg.RateLimitRequests, 5*time.Minute)

	return &Proxy{
		cfg:       cfg,
		rateLimit: rateLimiter,
		client:    client,
		logger:    logger,
		storage:   store,
		output:    os.Stdout,
	}
}

// Close 关闭代理
func (p *Proxy) Close() error {
	p.client.CloseIdleConnections()
	return nil
}

// ChatCompletions 聊天补全代理
func (p *Proxy) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	p.Forward(w, r)
}

// Forward 通用透传代理
func (p *Proxy) Forward(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	clientIP := getClientIP(r)

	// 速率限制检查
	if !p.rateLimit.Allow() {
		p.writeError(w, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
		return
	}

	// 检查是否需要模拟 /models 响应
	if p.cfg.MockModels && p.isModelsRequest(r.URL.Path) {
		p.handleMockModels(w, r, startTime, clientIP)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, p.cfg.MaxRequestBodySize))
	if err != nil {
		p.writeError(w, http.StatusBadRequest, "读取请求体失败")
		return
	}
	defer r.Body.Close()

	reqBody, model, inputTokens := parseRequestMetadata(body)

	// 检查请求是否要求流式响应
	isStreamRequest := false
	if reqBody != nil {
		if stream, ok := reqBody["stream"].(bool); ok {
			isStreamRequest = stream
		}
	}

	// 验证本地 API Key
	if !p.validateLocalAPIKey(r) {
		p.writeError(w, http.StatusUnauthorized, "API Key 无效")
		return
	}

	// 获取 Coding Plan API Key
	codingAPIKey := p.cfg.APIKey
	if codingAPIKey == "" {
		p.writeError(w, http.StatusInternalServerError, "服务未配置 API Key")
		return
	}

	// 获取服务商配置
	provider, err := p.cfg.GetProviderConfig()
	if err != nil {
		p.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 构建目标 URL
	baseURL := provider.CodingBaseURL
	if !p.cfg.UseCodingEndpoint {
		baseURL = provider.GeneralBaseURL
	}
	targetURL := buildTargetURL(baseURL, r, p.cfg.RemoveVersionPath)

	// 构建请求头
	headers := p.buildHeaders(provider, codingAPIKey, r.Header)

	// 日志记录
	p.logForwardRequest(model, inputTokens)

	// 插入待处理记录到数据库
	pendingRecord := &storage.RequestRecord{
		Timestamp:   startTime,
		Provider:    p.cfg.Provider,
		Model:       model,
		Method:      r.Method,
		Path:        r.URL.Path,
		ClientIP:    clientIP,
		RequestBody: string(body),
		InputTokens: inputTokens,
	}
	recordID, err := p.storage.InsertPendingRequest(pendingRecord)
	if err != nil {
		p.logger.Error("插入待处理记录失败", zap.Error(err))
	}

	// 辅助函数：异步更新记录为失败状态
	updateFailedRecord := func(statusCode int, errMsg string) {
		if recordID > 0 {
			go func() {
				duration := time.Since(startTime).Milliseconds()
				updateRecord := &storage.RequestRecord{
					StatusCode:  statusCode,
					Duration:    float64(duration),
					Success:     false,
					ErrorMsg:    errMsg,
				}
				if err := p.storage.UpdateRequestWithResponse(recordID, updateRecord); err != nil {
					p.logger.Error("更新失败记录失败", zap.Error(err))
				}
			}()
		}
	}

	// 创建上游请求
	upstreamReq, err := http.NewRequestWithContext(r.Context(), "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		updateFailedRecord(http.StatusInternalServerError, "创建请求失败")
		p.writeError(w, http.StatusInternalServerError, "创建请求失败")
		return
	}

	// 设置请求头
	for k, values := range headers {
		upstreamReq.Header[k] = append([]string(nil), values...)
	}

	// 发送请求
	resp, err := p.client.Do(upstreamReq)
	if err != nil {
		if strings.Contains(err.Error(), "context canceled") {
			updateFailedRecord(499, "请求被取消")
			p.logger.Info("请求被取消", zap.String("model", model))
			return
		}
		updateFailedRecord(http.StatusBadGateway, "上游服务不可用: "+err.Error())
		p.logger.Error("上游请求失败", zap.Error(err))
		p.writeError(w, http.StatusBadGateway, "上游服务不可用")
		return
	}
	defer resp.Body.Close()

	// 处理响应 - 方案A: 同时检查请求中的 stream 参数和响应 Content-Type
	// 如果客户端请求 stream=true，或者上游返回 SSE 格式，都使用流式处理
	isStreamResponse := isStreamRequest || isEventStream(resp.Header.Get("Content-Type"))
	if resp.StatusCode == http.StatusOK && isStreamResponse {
		p.handleStreamResponseWithStats(w, resp, startTime, r.Method, r.URL.Path, targetURL, model, clientIP, inputTokens, string(body), recordID)
	} else {
		p.handleNormalResponseWithStats(w, resp, startTime, r.Method, r.URL.Path, targetURL, model, clientIP, inputTokens, string(body), recordID)
	}
}

// Embeddings 向量嵌入代理
func (p *Proxy) Embeddings(w http.ResponseWriter, r *http.Request) {
	p.Forward(w, r)
}

// isModelsRequest 检查是否是 /models 请求
// 匹配规则:
// - 始终匹配 /models
// - 始终匹配 /v1/models, /v2/models, /v3/models (版本前缀格式)
func (p *Proxy) isModelsRequest(path string) bool {
	path = strings.TrimSuffix(path, "/")

	// 匹配 /models
	if path == "/models" {
		return true
	}

	// 匹配 /v1/models, /v2/models, /v3/models 等
	if strings.HasSuffix(path, "/models") {
		prefix := strings.TrimSuffix(path, "/models")
		return prefix == "/v1" || prefix == "/v2" || prefix == "/v3"
	}

	return false
}

// handleMockModels 处理模拟 /models 响应
func (p *Proxy) handleMockModels(w http.ResponseWriter, r *http.Request, startTime time.Time, clientIP string) {
	duration := time.Since(startTime).Milliseconds()

	// 验证本地 API Key
	if !p.validateLocalAPIKey(r) {
		p.writeError(w, http.StatusUnauthorized, "API Key 无效")
		return
	}

	// 返回模拟响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(p.cfg.MockModelsResp))

	// 打印日志
	p.logResponse(r.Method, r.URL.Path, "mock://models", http.StatusOK, duration, clientIP, p.cfg.MockModelsResp)
}

// validateLocalAPIKey 验证本地 API Key
func (p *Proxy) validateLocalAPIKey(r *http.Request) bool {
	localAPIKey := p.cfg.LocalAPIKey

	// 如果未配置本地 API Key，则不验证
	if localAPIKey == "" {
		return true
	}

	// 从请求头获取客户端 API Key
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// 移除 "Bearer " 前缀
	clientKey := strings.TrimPrefix(authHeader, "Bearer ")
	return clientKey == localAPIKey
}

// buildHeaders 构建请求头
func (p *Proxy) buildHeaders(provider *config.ProviderConfig, apiKey string, requestHeaders http.Header) http.Header {
	// 获取有效的 User-Agent（基于伪装工具配置）
	userAgent := p.cfg.GetEffectiveUserAgent()

	headers := make(http.Header, len(requestHeaders)+len(provider.ExtraHeaders)+2)
	for k, values := range requestHeaders {
		canonicalKey := textproto.CanonicalMIMEHeaderKey(k)
		if isHopByHopHeader(canonicalKey) {
			continue
		}
		if canonicalKey == "Authorization" || canonicalKey == textproto.CanonicalMIMEHeaderKey(provider.AuthHeader) {
			continue
		}
		headers[canonicalKey] = append([]string(nil), values...)
	}

	headers.Set(provider.AuthHeader, provider.AuthPrefix+apiKey)
	headers.Set("User-Agent", userAgent)

	for k, v := range p.cfg.GetDisguiseHeaders() {
		if headers.Get(k) == "" {
			headers.Set(k, v)
		}
	}

	// 添加额外头部
	for k, v := range provider.ExtraHeaders {
		headers.Set(k, v)
	}

	return headers
}

// handleStreamResponseWithStats 处理流式响应并统计
func (p *Proxy) handleStreamResponseWithStats(w http.ResponseWriter, resp *http.Response, startTime time.Time, method, path, targetURL, model, clientIP string, inputTokens int, requestBody string, recordID int64) {
	copyHeaders(w.Header(), resp.Header)

	// 设置 SSE 头
	if !isEventStream(w.Header().Get("Content-Type")) {
		w.Header().Set("Content-Type", "text/event-stream")
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 获取 flusher（在 WriteHeader 之前检查）
	flusher, ok := w.(http.Flusher)
	if !ok {
		p.writeError(w, http.StatusInternalServerError, "不支持流式响应")
		return
	}

	w.WriteHeader(resp.StatusCode)

	// 读取并转发响应，同时收集数据
	var responseBuf bytes.Buffer
	var outputTokens int
	var responseText strings.Builder

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		responseBuf.WriteString(line + "\n")

		if _, err := fmt.Fprintln(w, line); err != nil {
			p.logger.Warn("写入流式响应失败", zap.Error(err))
			break
		}
		flusher.Flush()

		// 解析 SSE 数据提取 token
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk map[string]interface{}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			responseText.WriteString(extractResponseText(chunk))

			// 提取 usage 中的 token
			if usage, ok := chunk["usage"].(map[string]interface{}); ok {
				if pt, ok := usage["total_tokens"].(float64); ok {
					outputTokens = int(pt)
				}
				if pt, ok := usage["completion_tokens"].(float64); ok {
					outputTokens = int(pt)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		p.logger.Warn("读取流式响应失败", zap.Error(err))
	}

	if outputTokens == 0 {
		outputTokens = estimateTextTokens(responseText.String())
	}

	duration := time.Since(startTime).Milliseconds()
	totalTokens := inputTokens + outputTokens

	// 打印响应日志
	p.logResponse(method, path, targetURL, resp.StatusCode, duration, clientIP, responseBuf.String())

	// 异步更新记录（不影响响应）
	if recordID > 0 {
		go func() {
			record := &storage.RequestRecord{
				ResponseBody: responseBuf.String(),
				StatusCode:   resp.StatusCode,
				Duration:     float64(duration),
				OutputTokens: outputTokens,
				TotalTokens:  totalTokens,
				Success:      resp.StatusCode == 200,
			}
			if err := p.storage.UpdateRequestWithResponse(recordID, record); err != nil {
				p.logger.Error("更新请求记录失败", zap.Error(err))
			}
		}()
	}
}

// handleNormalResponseWithStats 处理普通响应并统计
func (p *Proxy) handleNormalResponseWithStats(w http.ResponseWriter, resp *http.Response, startTime time.Time, method, path, targetURL, model, clientIP string, inputTokens int, requestBody string, recordID int64) {
	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Error("读取响应体失败", zap.Error(err))
		return
	}

	// 复制响应头
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	duration := time.Since(startTime).Milliseconds()

	// 解析响应获取 token
	var outputTokens int
	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err == nil {
		if usage, ok := respData["usage"].(map[string]interface{}); ok {
			if pt, ok := usage["total_tokens"].(float64); ok {
				outputTokens = int(pt)
			}
			if pt, ok := usage["completion_tokens"].(float64); ok {
				outputTokens = int(pt)
			}
		}
	}
	if outputTokens == 0 {
		outputTokens = estimateOutputTokensFromResponse(respData, respBody)
	}

	totalTokens := inputTokens + outputTokens

	// 打印响应日志
	p.logResponse(method, path, targetURL, resp.StatusCode, duration, clientIP, string(respBody))

	// 异步更新记录（不影响响应）
	if recordID > 0 {
		go func() {
			record := &storage.RequestRecord{
				ResponseBody: string(respBody),
				StatusCode:   resp.StatusCode,
				Duration:     float64(duration),
				OutputTokens: outputTokens,
				TotalTokens:  totalTokens,
				Success:      resp.StatusCode == 200,
			}
			if err := p.storage.UpdateRequestWithResponse(recordID, record); err != nil {
				p.logger.Error("更新请求记录失败", zap.Error(err))
			}
		}()
	}
}

// writeError 写入错误响应
func (p *Proxy) writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	errorResp := map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"type":    "proxy_error",
			"code":    fmt.Sprintf("%d", code),
		},
	}

	json.NewEncoder(w).Encode(errorResp)
}

// Stats 返回统计信息
func (p *Proxy) Stats() map[string]interface{} {
	count, max, remaining := p.rateLimit.Stats()
	stats, err := p.storage.GetStats()
	if err != nil || stats == nil {
		stats = &storage.Stats{}
	}
	return map[string]interface{}{
		"request_count":    count,
		"rate_limit":       max,
		"window_remaining": remaining.String(),
		"total_tokens":     stats.TotalTokens,
		"total_input":      stats.TotalInputTokens,
		"total_output":     stats.TotalOutputTokens,
	}
}

func copyHeaders(dst, src http.Header) {
	for k, values := range src {
		dst[k] = append([]string(nil), values...)
	}
}

func isEventStream(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

func buildTargetURL(baseURL string, r *http.Request, removeVersionPath bool) string {
	targetURL := strings.TrimRight(baseURL, "/")
	if r.URL.Path != "" {
		path := r.URL.Path
		// 如果启用了移除版本路径，则移除 /v1, /v2 等版本前缀
		if removeVersionPath {
			path = removeVersionPrefix(path)
		}
		if path != "" {
			targetURL += "/" + strings.TrimLeft(path, "/")
		}
	}
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}
	return targetURL
}

// versionPrefixRegex 匹配版本前缀的正则表达式
// 匹配: /v1, /v2, /v1beta, /v2alpha, /v3rc 等（可选带尾部斜杠）
var versionPrefixRegex = regexp.MustCompile(`^/?v\d+[a-z]*(?:/|$)`)

// removeVersionPrefix 移除路径中的版本前缀（如 /v1, /v2 等）
func removeVersionPrefix(path string) string {
	// 使用正则匹配：/v 后面跟数字，可选跟 alpha/beta/rc 等后缀
	// 如果匹配到，移除版本前缀部分
	if versionPrefixRegex.MatchString(path) {
		// 移除开头的 / 和版本号部分
		path = versionPrefixRegex.ReplaceAllString(path, "")
		return strings.Trim(path, "/")
	}
	return strings.Trim(path, "/")
}

func isHopByHopHeader(key string) bool {
	switch key {
	case "Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade":
		return true
	default:
		return false
	}
}

func parseRequestMetadata(body []byte) (map[string]interface{}, string, int) {
	if len(body) == 0 {
		return nil, "", 0
	}

	var reqBody map[string]interface{}
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return nil, "", 0
	}

	model, _ := reqBody["model"].(string)
	return reqBody, model, estimateInputTokens(reqBody)
}

func (p *Proxy) logForwardRequest(model string, inputTokens int) {
	fields := []zap.Field{
		zap.Int("input_tokens", inputTokens),
	}
	if model != "" {
		fields = append(fields, zap.String("model", model))
	}
	if p.cfg.Debug {
		fields = append(fields, zap.String("provider", p.cfg.Provider))
		p.logger.Info("处理请求", fields...)
		return
	}
	fmt.Fprintf(p.logOutput(), "时间：%s 转发请求：模型：%s token数：%d\n", humanLogTime(), displayModel(model), inputTokens)
}

func (p *Proxy) logForwardResponse(model string, outputTokens int) {
	fields := []zap.Field{
		zap.Int("output_tokens", outputTokens),
	}
	if model != "" {
		fields = append(fields, zap.String("model", model))
	}
	if p.cfg.Debug {
		p.logger.Info("请求完成", fields...)
		return
	}
	fmt.Fprintf(p.logOutput(), "时间：%s 转发响应：模型：%s token数：%d\n", humanLogTime(), displayModel(model), outputTokens)
}

// logResponse 打印响应日志
func (p *Proxy) logResponse(method, path, targetURL string, statusCode int, duration int64, clientIP, responseBody string) {
	// 判断是否是错误状态码 (4xx 或 5xx)
	isError := statusCode >= 400 && statusCode < 600

	fields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
		zap.String("target", targetURL),
		zap.Int("status", statusCode),
		zap.Int64("duration_ms", duration),
		zap.String("remote", clientIP),
	}

	if isError {
		// 限制响应体长度，避免日志过大
		truncatedBody := responseBody
		if len(truncatedBody) > 500 {
			truncatedBody = truncatedBody[:500] + "...(truncated)"
		}
		fields = append(fields, zap.String("response", truncatedBody))
		p.logger.Warn("代理响应", fields...)
	} else {
		p.logger.Info("代理响应", fields...)
	}
}

func (p *Proxy) logOutput() io.Writer {
	if p.output != nil {
		return p.output
	}
	return os.Stdout
}

func humanLogTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func displayModel(model string) string {
	if model == "" {
		return "-"
	}
	return model
}

func estimateOutputTokensFromResponse(respData map[string]interface{}, respBody []byte) int {
	if len(respData) != 0 {
		if tokens := estimateTextTokens(extractResponseText(respData)); tokens > 0 {
			return tokens
		}
	}
	return estimateTextTokens(string(respBody))
}

func estimateTextTokens(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	return utf8.RuneCountInString(text) / 2
}

func extractResponseText(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		var b strings.Builder
		for _, item := range v {
			b.WriteString(extractResponseText(item))
		}
		return b.String()
	case map[string]interface{}:
		priorityKeys := []string{"output_text", "content", "text", "message", "delta"}
		var b strings.Builder
		for _, key := range priorityKeys {
			if child, ok := v[key]; ok {
				b.WriteString(extractResponseText(child))
			}
		}
		if choices, ok := v["choices"]; ok {
			b.WriteString(extractResponseText(choices))
		}
		return b.String()
	default:
		return ""
	}
}

// getClientIP 获取客户端 IP
func getClientIP(r *http.Request) string {
	// 检查代理头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	return r.RemoteAddr
}

// estimateInputTokens 估算输入 token 数量
func estimateInputTokens(reqBody map[string]interface{}) int {
	// 简单估算：每个字符约 0.5 token
	if messages, ok := reqBody["messages"].([]interface{}); ok {
		totalChars := 0
		for _, msg := range messages {
			if m, ok := msg.(map[string]interface{}); ok {
				if content, ok := m["content"].(string); ok {
					totalChars += len(content)
				}
			}
		}
		return totalChars / 2 // 粗略估算
	}
	return 0
}
