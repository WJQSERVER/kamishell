package builtin

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"github.com/WJQSERVER-STUDIO/go-utils/iox"
	stdhttp "net/http"
	neturl "net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	httpc "github.com/WJQSERVER-STUDIO/httpc"
)

func init() {
	RegisterBuiltin(&BuiltinCommand{
		Name:        "http",
		Description: "发送 HTTP 请求",
		Usage:       "http [flags] [METHOD] URL",
		Help: `设计约定:
- 默认方法为 GET
- 使用 --data / --json / --form 时，默认方法自动切换为 POST
- 仅允许一种请求体模式：raw、json、form

常用示例:
  http "https://example.com/health"
  http "https://api.example.com/items" --json '{"name":"kami"}'
  http --headers "https://example.com"`,
		Action: HTTP,
	})
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

type optionalStringFlag struct {
	value string
	set   bool
}

func (f *optionalStringFlag) String() string {
	return f.value
}

func (f *optionalStringFlag) Set(value string) error {
	f.value = value
	f.set = true
	return nil
}

type httpCommandSpec struct {
	Request httpRequestSpec
	Client  httpClientSpec
	Output  httpOutputSpec
}

type httpRequestSpec struct {
	Method      string
	URL         string
	Headers     []httpHeader
	Query       []httpPair
	ContentType string
	Accept      string
	BasicAuth   *httpBasicAuth
	BearerToken string
	Body        httpBodySpec
}

type httpClientSpec struct {
	Timeout          time.Duration
	UserAgent        string
	NoDefaultHeaders bool
	Dump             bool
	Retry            httpRetrySpec
}

type httpRetrySpec struct {
	Count     int
	Statuses  []int
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

type httpOutputSpec struct {
	Mode        httpOutputMode
	OutputPath  string
	DiscardBody bool
}

type httpHeader struct {
	Key   string
	Value string
}

type httpPair struct {
	Key   string
	Value string
}

type httpBasicAuth struct {
	Username string
	Password string
}

type httpBodySpec struct {
	Kind   httpBodyKind
	Source *httpSourceSpec
	Form   []httpPair
}

type httpSourceSpec struct {
	Kind  httpSourceKind
	Value string
}

type httpBodyKind int

const (
	httpBodyNone httpBodyKind = iota
	httpBodyRaw
	httpBodyJSON
	httpBodyForm
)

type httpSourceKind int

const (
	httpSourceLiteral httpSourceKind = iota
	httpSourceFile
	httpSourceStdin
)

type httpOutputMode int

const (
	httpOutputBody httpOutputMode = iota
	httpOutputInclude
	httpOutputHeaders
	httpOutputStatus
)

func HTTP(args []string, env Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	_ = env
	if HandleBuiltinHelp(Builtins["http"], args, stdout) {
		return 0
	}

	spec, err := parseHTTPCommand(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "http: %v\n", err)
		return 1
	}

	client := spec.Client.newClient(stderr)
	req, err := spec.buildRequest(client, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "http: %v\n", err)
		return 1
	}

	resp, doErr := client.Do(req)
	if doErr != nil && resp == nil {
		fmt.Fprintf(stderr, "http: %v\n", doErr)
		return 1
	}
	defer resp.Body.Close()

	if err := spec.Output.write(stdout, resp); err != nil {
		fmt.Fprintf(stderr, "http: %v\n", err)
		return 1
	}

	if doErr != nil {
		fmt.Fprintf(stderr, "http: %v\n", doErr)
		return 1
	}
	if resp.StatusCode >= 400 {
		fmt.Fprintf(stderr, "http: unexpected status %s\n", resp.Status)
		return 1
	}

	return 0
}

func parseHTTPCommand(args []string, stderr io.Writer) (httpCommandSpec, error) {
	args = PreprocessArgs(args)
	args = normalizeHTTPArgsForFlagSet(args)

	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printHTTPUsage(stderr)
	}

	m := RegisterMeta("http")
	var methodFlag string
	var headers stringListFlag
	var queries stringListFlag
	var forms stringListFlag
	var dataFlag optionalStringFlag
	var jsonFlag optionalStringFlag
	var contentType string
	var accept string
	var auth string
	var bearer string
	var outputPath string
	var include bool
	var headersOnly bool
	var statusOnly bool
	var discardBody bool
	var timeout time.Duration
	var retries int
	var retryStatusCSV string
	var retryBase time.Duration
	var retryMax time.Duration
	var userAgent string
	var dump bool
	var noDefaultHeaders bool

	StringFlagVar(fs, m, &methodFlag, "method", "X", "", "HTTP method")
	m.SetFlagCompleter("method", func(cmdName string, argIndex int, prefix string) []string {
		return []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	})
	fs.Var(&headers, "H", "request header")
	fs.Var(&headers, "header", "request header")
	m.RegisterFlag("header", "H", "request header", FlagString)
	fs.Var(&queries, "q", "query parameter")
	fs.Var(&queries, "query", "query parameter")
	m.RegisterFlag("query", "q", "query parameter", FlagString)
	fs.Var(&forms, "f", "form field")
	fs.Var(&forms, "form", "form field")
	m.RegisterFlag("form", "f", "form field", FlagString)
	fs.Var(&dataFlag, "d", "raw request body")
	fs.Var(&dataFlag, "data", "raw request body")
	m.RegisterFlag("data", "d", "raw request body", FlagString)
	fs.Var(&jsonFlag, "j", "json request body")
	fs.Var(&jsonFlag, "json", "json request body")
	m.RegisterFlag("json", "j", "json request body", FlagString)
	StringFlagVar(fs, m, &contentType, "content-type", "", "", "request content type")
	StringFlagVar(fs, m, &accept, "accept", "", "", "request accept header")
	StringFlagVar(fs, m, &auth, "auth", "", "", "basic auth user:pass")
	StringFlagVar(fs, m, &bearer, "bearer", "", "", "bearer token")
	StringFlagVar(fs, m, &outputPath, "output", "o", "", "write response body to file")
	BoolFlagVar(fs, m, &include, "include", "i", false, "include response status and headers")
	BoolFlagVar(fs, m, &headersOnly, "headers", "I", false, "print response status and headers only")
	BoolFlagVar(fs, m, &statusOnly, "status", "s", false, "print response status only")
	BoolFlagVar(fs, m, &discardBody, "discard-body", "", false, "discard response body")
	DurationFlagVar(fs, m, &timeout, "timeout", "t", 0, "request timeout")
	IntFlagVar(fs, m, &retries, "retries", "r", 0, "retry count")
	StringFlagVar(fs, m, &retryStatusCSV, "retry-status", "", "429,500,502,503,504", "retry status codes")
	DurationFlagVar(fs, m, &retryBase, "retry-base", "", 100*time.Millisecond, "retry base delay")
	DurationFlagVar(fs, m, &retryMax, "retry-max", "", time.Second, "retry max delay")
	StringFlagVar(fs, m, &userAgent, "user-agent", "u", "", "user agent")
	BoolFlagVar(fs, m, &dump, "dump", "", false, "dump request log")
	BoolFlagVar(fs, m, &noDefaultHeaders, "no-default-headers", "", false, "disable default request headers")

	if err := fs.Parse(args); err != nil {
		return httpCommandSpec{}, err
	}

	request, err := buildHTTPRequestSpec(fs.Args(), httpRequestFlagValues{
		Method:      methodFlag,
		Headers:     headers,
		Query:       queries,
		Forms:       forms,
		Data:        dataFlag,
		JSON:        jsonFlag,
		ContentType: contentType,
		Accept:      accept,
		Auth:        auth,
		Bearer:      bearer,
	})
	if err != nil {
		return httpCommandSpec{}, err
	}

	output, err := buildHTTPOutputSpec(httpOutputFlagValues{
		Include:     include,
		HeadersOnly: headersOnly,
		StatusOnly:  statusOnly,
		DiscardBody: discardBody,
		OutputPath:  outputPath,
	})
	if err != nil {
		return httpCommandSpec{}, err
	}

	client, err := buildHTTPClientSpec(httpClientFlagValues{
		Timeout:          timeout,
		Retries:          retries,
		RetryStatusCSV:   retryStatusCSV,
		RetryBase:        retryBase,
		RetryMax:         retryMax,
		UserAgent:        userAgent,
		Dump:             dump,
		NoDefaultHeaders: noDefaultHeaders,
	})
	if err != nil {
		return httpCommandSpec{}, err
	}

	return httpCommandSpec{
		Request: request,
		Client:  client,
		Output:  output,
	}, nil
}

func normalizeHTTPArgsForFlagSet(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	valueFlags := map[string]bool{
		"-X":             true,
		"--method":       true,
		"-H":             true,
		"--header":       true,
		"-q":             true,
		"--query":        true,
		"-f":             true,
		"--form":         true,
		"-d":             true,
		"--data":         true,
		"-j":             true,
		"--json":         true,
		"--content-type": true,
		"--accept":       true,
		"--auth":         true,
		"--bearer":       true,
		"-o":             true,
		"--output":       true,
		"-t":             true,
		"--timeout":      true,
		"-r":             true,
		"--retries":      true,
		"--retry-status": true,
		"--retry-base":   true,
		"--retry-max":    true,
		"-u":             true,
		"--user-agent":   true,
	}

	boolFlags := map[string]bool{
		"-i":                   true,
		"--include":            true,
		"-I":                   true,
		"--headers":            true,
		"-s":                   true,
		"--status":             true,
		"--discard-body":       true,
		"--dump":               true,
		"--no-default-headers": true,
		"-h":                   true,
		"--help":               true,
	}

	flagArgs := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}

		name := arg
		hasInlineValue := false
		if strings.HasPrefix(arg, "--") {
			if before, _, ok := strings.Cut(arg, "="); ok {
				name = before
				hasInlineValue = true
			}
		}

		switch {
		case valueFlags[name]:
			flagArgs = append(flagArgs, arg)
			if hasInlineValue {
				continue
			}
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		case boolFlags[name]:
			flagArgs = append(flagArgs, arg)
		case strings.HasPrefix(arg, "-") && arg != "-":
			flagArgs = append(flagArgs, arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	return append(flagArgs, positionals...)
}

type httpRequestFlagValues struct {
	Method      string
	Headers     []string
	Query       []string
	Forms       []string
	Data        optionalStringFlag
	JSON        optionalStringFlag
	ContentType string
	Accept      string
	Auth        string
	Bearer      string
}

type httpClientFlagValues struct {
	Timeout          time.Duration
	Retries          int
	RetryStatusCSV   string
	RetryBase        time.Duration
	RetryMax         time.Duration
	UserAgent        string
	Dump             bool
	NoDefaultHeaders bool
}

type httpOutputFlagValues struct {
	Include     bool
	HeadersOnly bool
	StatusOnly  bool
	DiscardBody bool
	OutputPath  string
}

func buildHTTPRequestSpec(args []string, flags httpRequestFlagValues) (httpRequestSpec, error) {
	body, err := buildHTTPBodySpec(flags.Data, flags.JSON, flags.Forms)
	if err != nil {
		return httpRequestSpec{}, err
	}

	method, url, err := resolveHTTPMethodAndURL(args, flags.Method, body.hasContent())
	if err != nil {
		return httpRequestSpec{}, err
	}

	headers, err := parseHTTPHeaders(flags.Headers)
	if err != nil {
		return httpRequestSpec{}, err
	}

	query, err := parseHTTPPairs(flags.Query, "query")
	if err != nil {
		return httpRequestSpec{}, err
	}

	basicAuth, err := parseHTTPBasicAuth(flags.Auth)
	if err != nil {
		return httpRequestSpec{}, err
	}
	if basicAuth != nil && strings.TrimSpace(flags.Bearer) != "" {
		return httpRequestSpec{}, errors.New("--auth and --bearer cannot be used together")
	}

	return httpRequestSpec{
		Method:      method,
		URL:         url,
		Headers:     headers,
		Query:       query,
		ContentType: strings.TrimSpace(flags.ContentType),
		Accept:      strings.TrimSpace(flags.Accept),
		BasicAuth:   basicAuth,
		BearerToken: strings.TrimSpace(flags.Bearer),
		Body:        body,
	}, nil
}

func buildHTTPClientSpec(flags httpClientFlagValues) (httpClientSpec, error) {
	if flags.Retries < 0 {
		return httpClientSpec{}, errors.New("retries must be >= 0")
	}
	if flags.Timeout < 0 {
		return httpClientSpec{}, errors.New("timeout must be >= 0")
	}
	if flags.RetryBase < 0 || flags.RetryMax < 0 {
		return httpClientSpec{}, errors.New("retry delays must be >= 0")
	}

	statuses, err := parseRetryStatusList(flags.RetryStatusCSV)
	if err != nil {
		return httpClientSpec{}, err
	}

	return httpClientSpec{
		Timeout:          flags.Timeout,
		UserAgent:        strings.TrimSpace(flags.UserAgent),
		NoDefaultHeaders: flags.NoDefaultHeaders,
		Dump:             flags.Dump,
		Retry: httpRetrySpec{
			Count:     flags.Retries,
			Statuses:  statuses,
			BaseDelay: flags.RetryBase,
			MaxDelay:  flags.RetryMax,
		},
	}, nil
}

func buildHTTPOutputSpec(flags httpOutputFlagValues) (httpOutputSpec, error) {
	modeCount := 0
	if flags.Include {
		modeCount++
	}
	if flags.HeadersOnly {
		modeCount++
	}
	if flags.StatusOnly {
		modeCount++
	}
	if modeCount > 1 {
		return httpOutputSpec{}, errors.New("only one of --include, --headers or --status can be used")
	}

	mode := httpOutputBody
	if flags.Include {
		mode = httpOutputInclude
	}
	if flags.HeadersOnly {
		mode = httpOutputHeaders
	}
	if flags.StatusOnly {
		mode = httpOutputStatus
	}

	outputPath := strings.TrimSpace(flags.OutputPath)
	if outputPath != "" {
		if mode == httpOutputHeaders || mode == httpOutputStatus || flags.DiscardBody {
			return httpOutputSpec{}, errors.New("--output requires the response body to be enabled")
		}
	}

	return httpOutputSpec{
		Mode:        mode,
		OutputPath:  outputPath,
		DiscardBody: flags.DiscardBody,
	}, nil
}

func buildHTTPBodySpec(dataFlag optionalStringFlag, jsonFlag optionalStringFlag, forms []string) (httpBodySpec, error) {
	bodyModes := 0
	if dataFlag.set {
		bodyModes++
	}
	if jsonFlag.set {
		bodyModes++
	}
	if len(forms) > 0 {
		bodyModes++
	}
	if bodyModes > 1 {
		return httpBodySpec{}, errors.New("only one body mode can be used among --data, --json and --form")
	}

	switch {
	case dataFlag.set:
		return httpBodySpec{Kind: httpBodyRaw, Source: parseHTTPSourceSpec(dataFlag.value)}, nil
	case jsonFlag.set:
		return httpBodySpec{Kind: httpBodyJSON, Source: parseHTTPSourceSpec(jsonFlag.value)}, nil
	case len(forms) > 0:
		pairs, err := parseHTTPPairs(forms, "form field")
		if err != nil {
			return httpBodySpec{}, err
		}
		return httpBodySpec{Kind: httpBodyForm, Form: pairs}, nil
	default:
		return httpBodySpec{Kind: httpBodyNone}, nil
	}
}

func parseHTTPSourceSpec(value string) *httpSourceSpec {
	if strings.HasPrefix(value, "@@") {
		return &httpSourceSpec{Kind: httpSourceLiteral, Value: value[1:]}
	}
	if value == "-" {
		return &httpSourceSpec{Kind: httpSourceStdin, Value: "-"}
	}
	if strings.HasPrefix(value, "@") {
		return &httpSourceSpec{Kind: httpSourceFile, Value: value[1:]}
	}
	return &httpSourceSpec{Kind: httpSourceLiteral, Value: value}
}

func parseHTTPHeaders(values []string) ([]httpHeader, error) {
	headers := make([]httpHeader, 0, len(values))
	for _, value := range values {
		key, val, err := splitHeaderArgument(value)
		if err != nil {
			return nil, err
		}
		headers = append(headers, httpHeader{Key: key, Value: val})
	}
	return headers, nil
}

func parseHTTPPairs(values []string, label string) ([]httpPair, error) {
	pairs := make([]httpPair, 0, len(values))
	for _, value := range values {
		key, val, err := splitKeyValueArgument(value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s %q", label, value)
		}
		pairs = append(pairs, httpPair{Key: key, Value: val})
	}
	return pairs, nil
}

func parseHTTPBasicAuth(value string) (*httpBasicAuth, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("--auth expects user:pass")
	}
	return &httpBasicAuth{Username: parts[0], Password: parts[1]}, nil
}

func (s httpClientSpec) newClient(stderr io.Writer) *httpc.Client {
	opts := []httpc.Option{
		httpc.WithRetryOptions(httpc.RetryOptions{
			MaxAttempts:   s.Retry.Count,
			BaseDelay:     s.Retry.BaseDelay,
			MaxDelay:      s.Retry.MaxDelay,
			RetryStatuses: s.Retry.Statuses,
		}),
	}

	if s.Timeout > 0 {
		opts = append(opts, httpc.WithTimeout(s.Timeout))
	}
	if s.UserAgent != "" {
		opts = append(opts, httpc.WithUserAgent(s.UserAgent))
	}
	if s.Dump {
		opts = append(opts, httpc.WithDumpLogFunc(func(_ context.Context, log string) {
			_, _ = io.WriteString(stderr, log)
			if !strings.HasSuffix(log, "\n") {
				_, _ = io.WriteString(stderr, "\n")
			}
		}))
	}

	return httpc.New(opts...)
}

func (s httpCommandSpec) buildRequest(client *httpc.Client, stdin io.Reader) (*stdhttp.Request, error) {
	builder := client.NewRequestBuilder(s.Request.Method, s.Request.URL)
	if s.Client.NoDefaultHeaders {
		builder.NoDefaultHeaders()
	}
	for _, query := range s.Request.Query {
		builder.AddQueryParam(query.Key, query.Value)
	}

	body, autoContentType, err := s.Request.Body.read(stdin)
	if err != nil {
		return nil, err
	}
	if body != nil {
		builder.SetRawBody(body)
	}

	req, err := builder.Build()
	if err != nil {
		return nil, err
	}

	s.applyRequestMetadata(req, autoContentType)
	applyHTTPHeadersToRequest(req.Header, s.Request.Headers)
	ensureReplayableHTTPRequest(req)
	return req, nil
}

func (s httpCommandSpec) applyRequestMetadata(req *stdhttp.Request, autoContentType string) {
	if req == nil {
		return
	}
	if s.Request.Accept != "" {
		req.Header.Set("Accept", s.Request.Accept)
	}
	if autoContentType != "" {
		req.Header.Set("Content-Type", autoContentType)
	}
	if s.Request.ContentType != "" {
		req.Header.Set("Content-Type", s.Request.ContentType)
	}
	if s.Request.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.Request.BearerToken)
	}
	if s.Request.BasicAuth != nil {
		req.SetBasicAuth(s.Request.BasicAuth.Username, s.Request.BasicAuth.Password)
	}
}

func applyHTTPHeadersToRequest(dst stdhttp.Header, headers []httpHeader) {
	seen := make(map[string]bool, len(headers))
	for _, header := range headers {
		canonical := stdhttp.CanonicalHeaderKey(header.Key)
		if !seen[canonical] {
			dst.Del(canonical)
			seen[canonical] = true
		}
		dst.Add(header.Key, header.Value)
	}
}

func (s httpBodySpec) hasContent() bool {
	return s.Kind != httpBodyNone
}

func (s httpBodySpec) read(stdin io.Reader) ([]byte, string, error) {
	switch s.Kind {
	case httpBodyNone:
		return nil, "", nil
	case httpBodyRaw:
		body, err := s.Source.read(stdin)
		return body, "", err
	case httpBodyJSON:
		body, err := s.Source.read(stdin)
		return body, "application/json", err
	case httpBodyForm:
		values := make(neturl.Values)
		for _, pair := range s.Form {
			values.Add(pair.Key, pair.Value)
		}
		return []byte(values.Encode()), "application/x-www-form-urlencoded", nil
	default:
		return nil, "", errors.New("unsupported body mode")
	}
}

func (s *httpSourceSpec) read(stdin io.Reader) ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	switch s.Kind {
	case httpSourceLiteral:
		return []byte(s.Value), nil
	case httpSourceFile:
		if strings.TrimSpace(s.Value) == "" {
			return nil, errors.New("empty @file body reference")
		}
		body, err := os.ReadFile(s.Value)
		if err != nil {
			return nil, fmt.Errorf("read body file %s: %w", s.Value, err)
		}
		return body, nil
	case httpSourceStdin:
		if stdin == nil {
			return nil, errors.New("stdin body requested but stdin is unavailable")
		}
		return io.ReadAll(stdin)
	default:
		return nil, errors.New("unsupported body source")
	}
}

func (s httpOutputSpec) write(stdout io.Writer, resp *stdhttp.Response) error {
	if resp == nil {
		return errors.New("no response")
	}

	switch s.Mode {
	case httpOutputStatus:
		writeHTTPStatus(stdout, resp)
		return discardHTTPBody(resp.Body)
	case httpOutputHeaders:
		writeHTTPResponseMetadata(stdout, resp)
		return discardHTTPBody(resp.Body)
	case httpOutputInclude:
		writeHTTPResponseMetadata(stdout, resp)
		if s.DiscardBody {
			return discardHTTPBody(resp.Body)
		}
		fmt.Fprintln(stdout)
		return copyHTTPBody(stdout, s.OutputPath, resp.Body)
	default:
		if s.DiscardBody {
			return discardHTTPBody(resp.Body)
		}
		return copyHTTPBody(stdout, s.OutputPath, resp.Body)
	}
}

func copyHTTPBody(stdout io.Writer, outputPath string, body io.Reader) error {
	bodyWriter := stdout
	var outputFile *os.File
	var err error
	if outputPath != "" {
		outputFile, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("open output file %s: %w", outputPath, err)
		}
		defer outputFile.Close()
		bodyWriter = outputFile
	}
	if _, err := iox.Copy(bodyWriter, body); err != nil {
		return fmt.Errorf("copy response body: %w", err)
	}
	return nil
}

func discardHTTPBody(body io.Reader) error {
	if body == nil {
		return nil
	}
	_, err := iox.Copy(io.Discard, body)
	return err
}

func ensureReplayableHTTPRequest(req *stdhttp.Request) {
	if req == nil || req.GetBody != nil {
		return
	}
	if req.Body != nil && req.Body != stdhttp.NoBody {
		return
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("")), nil
	}
	req.Body = stdhttp.NoBody
}

func resolveHTTPMethodAndURL(args []string, methodFlag string, hasBody bool) (string, string, error) {
	flagMethod := strings.ToUpper(strings.TrimSpace(methodFlag))
	positionalMethod, url, hasPositionalMethod, err := parseHTTPPositionalTarget(args)
	if err != nil {
		return "", "", err
	}
	if flagMethod != "" && hasPositionalMethod {
		return "", "", errors.New("method specified both positionally and by flag")
	}
	if flagMethod != "" {
		return flagMethod, url, nil
	}
	if hasPositionalMethod {
		return positionalMethod, url, nil
	}
	return defaultHTTPMethod(hasBody), url, nil
}

func parseHTTPPositionalTarget(args []string) (string, string, bool, error) {
	switch len(args) {
	case 0:
		return "", "", false, errors.New("missing URL")
	case 1:
		return "", args[0], false, nil
	case 2:
		if !isCommonHTTPMethod(args[0]) {
			return "", "", false, fmt.Errorf("unexpected extra argument %q", args[1])
		}
		return strings.ToUpper(args[0]), args[1], true, nil
	default:
		return "", "", false, fmt.Errorf("too many arguments: %s", strings.Join(args[2:], " "))
	}
}

func defaultHTTPMethod(hasBody bool) string {
	if hasBody {
		return stdhttp.MethodPost
	}
	return stdhttp.MethodGet
}

func isCommonHTTPMethod(value string) bool {
	switch strings.ToUpper(value) {
	case stdhttp.MethodGet, stdhttp.MethodPost, stdhttp.MethodPut, stdhttp.MethodPatch, stdhttp.MethodDelete, stdhttp.MethodHead, stdhttp.MethodOptions:
		return true
	default:
		return false
	}
}

func parseRetryStatusList(csv string) ([]int, error) {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil, nil
	}

	parts := strings.Split(csv, ",")
	statuses := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		status, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid retry status %q", part)
		}
		if status < 100 || status > 599 {
			return nil, fmt.Errorf("retry status %d out of range", status)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func splitHeaderArgument(input string) (string, string, error) {
	colon := strings.Index(input, ":")
	equals := strings.Index(input, "=")
	idx := -1
	switch {
	case colon >= 0 && equals >= 0:
		idx = min(colon, equals)
	case colon >= 0:
		idx = colon
	case equals >= 0:
		idx = equals
	}
	if idx < 0 {
		return "", "", fmt.Errorf("invalid header %q", input)
	}

	key := strings.TrimSpace(input[:idx])
	value := strings.TrimSpace(input[idx+1:])
	if key == "" {
		return "", "", fmt.Errorf("invalid header %q", input)
	}
	return key, value, nil
}

func splitKeyValueArgument(input string) (string, string, error) {
	before, after, ok := strings.Cut(input, "=")
	if !ok {
		return "", "", errors.New("missing '=' separator")
	}
	key := strings.TrimSpace(before)
	value := strings.TrimSpace(after)
	if key == "" {
		return "", "", errors.New("missing key")
	}
	return key, value, nil
}

func writeHTTPStatus(w io.Writer, resp *stdhttp.Response) {
	fmt.Fprintf(w, "%s %s\n", resp.Proto, resp.Status)
}

func writeHTTPResponseMetadata(w io.Writer, resp *stdhttp.Response) {
	writeHTTPStatus(w, resp)
	keys := make([]string, 0, len(resp.Header))
	for key := range resp.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(w, "%s: %s\n", key, strings.Join(resp.Header.Values(key), ", "))
	}
}

func printHTTPUsage(w io.Writer) {
	fmt.Fprintln(w, "用法: http [flags] [METHOD] URL")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "请求语义:")
	fmt.Fprintln(w, "  - 未显式指定方法时，默认使用 GET")
	fmt.Fprintln(w, "  - 当使用 --data / --json / --form 时，默认方法自动切换为 POST")
	fmt.Fprintln(w, "  - 仅允许一种请求体模式：raw、json 或 form")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "请求参数:")
	fmt.Fprintln(w, "  -X, --method <method>         指定请求方法")
	fmt.Fprintln(w, "  -H, --header <k:v>            添加请求头，可重复")
	fmt.Fprintln(w, "  -q, --query <k=v>             添加查询参数，可重复")
	fmt.Fprintln(w, "  -d, --data <value|@file|->    原始请求体")
	fmt.Fprintln(w, "  -j, --json <value|@file|->    JSON 请求体，自动设置 Content-Type")
	fmt.Fprintln(w, "  -f, --form <k=v>              表单字段，可重复")
	fmt.Fprintln(w, "      --content-type <mime>     显式设置 Content-Type")
	fmt.Fprintln(w, "      --accept <mime>           设置 Accept")
	fmt.Fprintln(w, "      --auth <user:pass>        Basic Auth")
	fmt.Fprintln(w, "      --bearer <token>          Bearer Token")
	fmt.Fprintln(w, "  -u, --user-agent <ua>         自定义 User-Agent")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "执行控制:")
	fmt.Fprintln(w, "  -t, --timeout <duration>      请求超时，例如 5s")
	fmt.Fprintln(w, "  -r, --retries <n>             重试次数")
	fmt.Fprintln(w, "      --retry-status <codes>    重试状态码，例如 429,500,502")
	fmt.Fprintln(w, "      --retry-base <duration>   重试基础退避时间")
	fmt.Fprintln(w, "      --retry-max <duration>    重试最大退避时间")
	fmt.Fprintln(w, "      --dump                    输出请求日志")
	fmt.Fprintln(w, "      --no-default-headers      禁用默认请求头")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "响应输出:")
	fmt.Fprintln(w, "  -i, --include                 输出状态行、响应头和响应体")
	fmt.Fprintln(w, "  -I, --headers                 仅输出状态行和响应头")
	fmt.Fprintln(w, "  -s, --status                  仅输出状态行")
	fmt.Fprintln(w, "  -o, --output <file>           将响应体写入文件")
	fmt.Fprintln(w, "      --discard-body            丢弃响应体")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "示例:")
	fmt.Fprintln(w, "  http \"https://example.com/health\"")
	fmt.Fprintln(w, "  http POST \"https://api.example.com/items\" --json \"{\\\"name\\\":\\\"kami\\\"}\"")
	fmt.Fprintln(w, "  http \"https://api.example.com/items\" --form name=kami --form lang=zh")
	fmt.Fprintln(w, "  print \"{\\\"name\\\":\\\"kami\\\"}\" | http --method POST --header \"Content-Type: application/json\" --data - \"https://api.example.com/items\"")
	fmt.Fprintln(w, "  http -I \"https://example.com\"")
}
