// Command postprocess-generated normalizes Proto-declared HTTP behavior that
// the pinned upstream generators do not yet preserve consistently. It runs as
// the final deterministic step of `make generate`.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	annotationsv1 "github.com/multica-ai/multica/server/gen/go/annotations/v1"
	_ "github.com/multica-ai/multica/server/gen/go/auth/v1"
	_ "github.com/multica-ai/multica/server/gen/go/space/v1"
	_ "github.com/multica-ai/multica/server/gen/go/system/v1"
	_ "github.com/multica-ai/multica/server/gen/go/workspace/v1"
	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type statusOverride struct {
	protoPath   string
	serviceName string
	methodName  string
	status      int
}

type preBodyAuthorizationOverride struct {
	protoPath   string
	serviceName string
	methodName  string
	requestName string
}

type responseBodyOverride struct {
	serviceName           string
	methodName            string
	schemaName            string
	protoPath             string
	responseName          string
	responseJSONField     string
	responseGoField       string
	optionalJSONFields    []string
	optionalOpenAPIFields []string
}

func main() {
	overrides, err := declaredStatusOverrides()
	if err != nil {
		fatal(err)
	}
	for _, override := range overrides {
		path := filepath.Join("gen/go", strings.TrimSuffix(override.protoPath, ".proto")+"_http.pb.go")
		if err := rewriteFile(path, func(input []byte) ([]byte, error) {
			return applyGoHTTPStatus(input, override)
		}); err != nil {
			fatal(err)
		}
	}
	preBodyAuthorizations, err := declaredPreBodyAuthorizations()
	if err != nil {
		fatal(err)
	}
	for _, override := range preBodyAuthorizations {
		path := filepath.Join("gen/go", strings.TrimSuffix(override.protoPath, ".proto")+"_http.pb.go")
		if err := rewriteFile(path, func(input []byte) ([]byte, error) {
			return applyGoHTTPPreBodyAuthorization(input, override)
		}); err != nil {
			fatal(err)
		}
	}
	if err := rewriteGeneratedHTTPClients(filepath.Join("gen", "go")); err != nil {
		fatal(err)
	}
	responseBodies, err := declaredResponseBodyOverrides()
	if err != nil {
		fatal(err)
	}
	if err := rewriteGeneratedResponseBodyJSONTags(filepath.Join("gen", "go"), responseBodies); err != nil {
		fatal(err)
	}
	if err := rewriteGeneratedHTTPResponseBodyEmptyArrays(filepath.Join("gen", "go"), responseBodies); err != nil {
		fatal(err)
	}
	if err := rewriteGeneratedContractResponseBodyJSONTags(filepath.Join("internal", "modules"), responseBodies); err != nil {
		fatal(err)
	}
	openAPIPath := filepath.Join("gen", "openapi", "openapi.yaml")
	if err := rewriteFile(openAPIPath, func(input []byte) ([]byte, error) {
		output, transformErr := applyOpenAPIStatuses(input, overrides)
		if transformErr != nil {
			return nil, transformErr
		}
		return applyOpenAPIResponseBodies(output, responseBodies)
	}); err != nil {
		fatal(err)
	}
}

func declaredPreBodyAuthorizations() ([]preBodyAuthorizationOverride, error) {
	var overrides []preBodyAuthorizationOverride
	var declarationErr error
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for serviceIndex := 0; serviceIndex < services.Len(); serviceIndex++ {
			service := services.Get(serviceIndex)
			methods := service.Methods()
			for methodIndex := 0; methodIndex < methods.Len(); methodIndex++ {
				method := methods.Get(methodIndex)
				options, ok := method.Options().(*descriptorpb.MethodOptions)
				if !ok || !proto.HasExtension(options, annotationsv1.E_AuthorizeBeforeBody) {
					continue
				}
				enabled, ok := proto.GetExtension(options, annotationsv1.E_AuthorizeBeforeBody).(bool)
				if !ok {
					declarationErr = fmt.Errorf("invalid authorize_before_body option on %s.%s", service.Name(), method.Name())
					return false
				}
				if enabled {
					overrides = append(overrides, preBodyAuthorizationOverride{
						protoPath: file.Path(), serviceName: string(service.Name()),
						methodName: string(method.Name()), requestName: string(method.Input().Name()),
					})
				}
			}
		}
		return true
	})
	return overrides, declarationErr
}

func applyGoHTTPPreBodyAuthorization(input []byte, override preBodyAuthorizationOverride) ([]byte, error) {
	output := string(input)
	interfaceStart := strings.Index(output, "type "+override.serviceName+"HTTPServer interface {")
	if interfaceStart < 0 {
		return nil, fmt.Errorf("HTTP server interface not found for %s", override.serviceName)
	}
	interfaceEndOffset := strings.Index(output[interfaceStart:], "\n}")
	if interfaceEndOffset < 0 {
		return nil, fmt.Errorf("HTTP server interface is incomplete for %s", override.serviceName)
	}
	interfaceEnd := interfaceStart + interfaceEndOffset
	hookName := "Authorize" + override.methodName
	hookDeclaration := "\t" + hookName + "(context.Context, *" + override.requestName + ") error"
	output = output[:interfaceEnd] + "\n" + hookDeclaration + output[interfaceEnd:]

	functionPrefix := "func _" + override.serviceName + "_" + override.methodName
	handlerStart := strings.Index(output, functionPrefix)
	if handlerStart < 0 {
		return nil, fmt.Errorf("HTTP handler not found for %s.%s", override.serviceName, override.methodName)
	}
	handlerEndOffset := strings.Index(output[handlerStart:], "\n}\n")
	if handlerEndOffset < 0 {
		return nil, fmt.Errorf("HTTP handler is incomplete for %s.%s", override.serviceName, override.methodName)
	}
	handlerEnd := handlerStart + handlerEndOffset + len("\n}\n")
	segment := output[handlerStart:handlerEnd]
	bodyThenVars := "\t\tif err := ctx.Bind(&in); err != nil {\n\t\t\treturn err\n\t\t}\n" +
		"\t\tif err := ctx.BindVars(&in); err != nil {\n\t\t\treturn err\n\t\t}"
	varsOnly := "\t\tif err := ctx.BindVars(&in); err != nil {\n\t\t\treturn err\n\t\t}"
	if !strings.Contains(segment, bodyThenVars) {
		return nil, fmt.Errorf("body/path binding block not found for %s.%s", override.serviceName, override.methodName)
	}
	segment = strings.Replace(segment, bodyThenVars, varsOnly, 1)
	generatedMiddleware := "\t\th := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {\n" +
		"\t\t\treturn srv." + override.methodName + "(ctx, req.(*" + override.requestName + "))\n\t\t})"
	authorizedMiddleware := "\t\th := ctx.Middleware(func(callCtx context.Context, req interface{}) (interface{}, error) {\n" +
		"\t\t\tif err := srv." + hookName + "(callCtx, &in); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n" +
		"\t\t\tif err := ctx.Bind(&in); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n" +
		"\t\t\tif err := ctx.BindVars(&in); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n" +
		"\t\t\treturn srv." + override.methodName + "(callCtx, req.(*" + override.requestName + "))\n\t\t})"
	if !strings.Contains(segment, generatedMiddleware) {
		return nil, fmt.Errorf("middleware block not found for %s.%s", override.serviceName, override.methodName)
	}
	segment = strings.Replace(segment, generatedMiddleware, authorizedMiddleware, 1)
	output = output[:handlerStart] + segment + output[handlerEnd:]
	return []byte(output), nil
}

func declaredResponseBodyOverrides() ([]responseBodyOverride, error) {
	var overrides []responseBodyOverride
	var declarationErr error
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for serviceIndex := 0; serviceIndex < services.Len(); serviceIndex++ {
			service := services.Get(serviceIndex)
			methods := service.Methods()
			for methodIndex := 0; methodIndex < methods.Len(); methodIndex++ {
				override, err := responseBodyOverrideForMethod(file, service, methods.Get(methodIndex))
				if err != nil {
					declarationErr = err
					return false
				}
				if override != nil {
					overrides = append(overrides, *override)
				}
			}
		}
		return declarationErr == nil
	})
	return overrides, declarationErr
}

func responseBodyOverrideForMethod(
	file protoreflect.FileDescriptor,
	service protoreflect.ServiceDescriptor,
	method protoreflect.MethodDescriptor,
) (*responseBodyOverride, error) {
	options, ok := method.Options().(*descriptorpb.MethodOptions)
	if !ok || !proto.HasExtension(options, annotations.E_Http) {
		return nil, nil
	}
	rule, ok := proto.GetExtension(options, annotations.E_Http).(*annotations.HttpRule)
	if !ok || rule.GetResponseBody() == "" || rule.GetResponseBody() == "*" {
		return nil, nil
	}
	field := method.Output().Fields().ByName(protoreflect.Name(rule.GetResponseBody()))
	if field == nil || !field.IsList() || field.Kind() != protoreflect.MessageKind {
		return nil, fmt.Errorf("unsupported HTTP response_body %s on %s.%s", rule.GetResponseBody(), service.Name(), method.Name())
	}
	message := field.Message()
	optionalJSONFields := make([]string, 0)
	optionalOpenAPIFields := make([]string, 0)
	for fieldIndex := 0; fieldIndex < message.Fields().Len(); fieldIndex++ {
		messageField := message.Fields().Get(fieldIndex)
		if messageField.HasOptionalKeyword() {
			optionalJSONFields = append(optionalJSONFields, string(messageField.Name()))
			optionalOpenAPIFields = append(optionalOpenAPIFields, messageField.JSONName())
		}
	}
	return &responseBodyOverride{
		serviceName:           string(service.Name()),
		methodName:            string(method.Name()),
		schemaName:            string(message.Name()),
		protoPath:             file.Path(),
		responseName:          string(method.Output().Name()),
		responseJSONField:     field.JSONName(),
		responseGoField:       snakeToUpperCamel(string(field.Name())),
		optionalJSONFields:    optionalJSONFields,
		optionalOpenAPIFields: optionalOpenAPIFields,
	}, nil
}

func rewriteGeneratedResponseBodyJSONTags(root string, overrides []responseBodyOverride) error {
	for _, override := range overrides {
		if len(override.optionalJSONFields) == 0 {
			continue
		}
		path := filepath.Join(root, strings.TrimSuffix(override.protoPath, ".proto")+".pb.go")
		if err := rewriteFile(path, func(input []byte) ([]byte, error) {
			return applyGoResponseBodyOptionalFields(input, override)
		}); err != nil {
			return err
		}
	}
	return nil
}

func applyGoResponseBodyOptionalFields(input []byte, override responseBodyOverride) ([]byte, error) {
	lines := strings.Split(string(input), "\n")
	structStart := findTrimmedLine(lines, "type "+override.schemaName+" struct {", 0)
	if structStart < 0 {
		return nil, fmt.Errorf("generated response_body message %s not found", override.schemaName)
	}
	structEnd := findTrimmedLine(lines, "}", structStart+1)
	if structEnd < 0 {
		return nil, fmt.Errorf("generated response_body message %s is incomplete", override.schemaName)
	}
	for _, fieldName := range override.optionalJSONFields {
		from := `json:"` + fieldName + `,omitempty"`
		to := `json:"` + fieldName + `"`
		found := false
		for index := structStart + 1; index < structEnd; index++ {
			if strings.Contains(lines[index], from) {
				lines[index] = strings.Replace(lines[index], from, to, 1)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("optional JSON field %s.%s not found", override.schemaName, fieldName)
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func rewriteGeneratedContractResponseBodyJSONTags(root string, overrides []responseBodyOverride) error {
	for _, override := range overrides {
		moduleName := strings.SplitN(override.protoPath, "/", 2)[0]
		serviceBase := strings.TrimSuffix(override.serviceName, "Service")
		path := filepath.Join(root, moduleName, "contract", camelToSnake(serviceBase)+"_service.go")
		if err := rewriteFile(path, func(input []byte) ([]byte, error) {
			return applyGoContractResponseBodyField(input, override)
		}); err != nil {
			return err
		}
	}
	return nil
}

func applyGoContractResponseBodyField(input []byte, override responseBodyOverride) ([]byte, error) {
	serviceBase := strings.TrimSuffix(override.serviceName, "Service")
	typeName := serviceBase + "_" + override.responseName
	lines := strings.Split(string(input), "\n")
	structStart := findTrimmedLine(lines, "type "+typeName+" struct {", 0)
	if structStart < 0 {
		return nil, fmt.Errorf("generated response_body contract %s not found", typeName)
	}
	structEnd := findTrimmedLine(lines, "}", structStart+1)
	if structEnd < 0 {
		return nil, fmt.Errorf("generated response_body contract %s is incomplete", typeName)
	}
	from := `json:"` + override.responseJSONField + `,omitempty"`
	to := `json:"` + override.responseJSONField + `"`
	for index := structStart + 1; index < structEnd; index++ {
		if strings.Contains(lines[index], from) {
			lines[index] = strings.Replace(lines[index], from, to, 1)
			return []byte(strings.Join(lines, "\n")), nil
		}
	}
	return nil, fmt.Errorf("response_body contract field %s.%s not found", typeName, override.responseJSONField)
}

func camelToSnake(input string) string {
	var output strings.Builder
	for index, character := range input {
		if index > 0 && character >= 'A' && character <= 'Z' {
			_ = output.WriteByte('_')
		}
		_, _ = output.WriteRune(character)
	}
	return strings.ToLower(output.String())
}

func snakeToUpperCamel(input string) string {
	value := snakeToLowerCamel(input)
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func rewriteGeneratedHTTPResponseBodyEmptyArrays(root string, overrides []responseBodyOverride) error {
	for _, override := range overrides {
		path := filepath.Join(root, strings.TrimSuffix(override.protoPath, ".proto")+"_http.pb.go")
		if err := rewriteFile(path, func(input []byte) ([]byte, error) {
			return applyGoHTTPResponseBodyEmptyArray(input, override)
		}); err != nil {
			return err
		}
	}
	return nil
}

func applyGoHTTPResponseBodyEmptyArray(input []byte, override responseBodyOverride) ([]byte, error) {
	functionPrefix := "func _" + override.serviceName + "_" + override.methodName
	start := strings.Index(string(input), functionPrefix)
	if start < 0 {
		return nil, fmt.Errorf("response_body handler not found for %s.%s", override.serviceName, override.methodName)
	}
	relativeEnd := strings.Index(string(input[start:]), "\n}\n")
	if relativeEnd < 0 {
		return nil, fmt.Errorf("unterminated response_body handler for %s.%s", override.serviceName, override.methodName)
	}
	end := start + relativeEnd + len("\n}\n")
	segment := string(input[start:end])
	resultLine := "\t\treturn ctx.Result(200, reply." + override.responseGoField + ")"
	if !strings.Contains(segment, resultLine) {
		return nil, fmt.Errorf("response_body result not found for %s.%s", override.serviceName, override.methodName)
	}
	normalized := "\t\tif reply." + override.responseGoField + " == nil {\n" +
		"\t\t\treply." + override.responseGoField + " = []*" + override.schemaName + "{}\n" +
		"\t\t}\n" + resultLine
	segment = strings.Replace(segment, resultLine, normalized, 1)
	return append(append([]byte(nil), input[:start]...), append([]byte(segment), input[end:]...)...), nil
}

func declaredStatusOverrides() ([]statusOverride, error) {
	var overrides []statusOverride
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for serviceIndex := 0; serviceIndex < services.Len(); serviceIndex++ {
			service := services.Get(serviceIndex)
			methods := service.Methods()
			for methodIndex := 0; methodIndex < methods.Len(); methodIndex++ {
				method := methods.Get(methodIndex)
				options, ok := method.Options().(*descriptorpb.MethodOptions)
				if !ok || !proto.HasExtension(options, annotationsv1.E_HttpSuccessStatus) {
					continue
				}
				status, ok := proto.GetExtension(options, annotationsv1.E_HttpSuccessStatus).(int32)
				if !ok || status < 200 || status > 299 {
					overrides = nil
					return false
				}
				overrides = append(overrides, statusOverride{
					protoPath:   file.Path(),
					serviceName: string(service.Name()),
					methodName:  string(method.Name()),
					status:      int(status),
				})
			}
		}
		return true
	})
	if overrides == nil {
		return nil, errors.New("invalid annotations.v1.http_success_status option")
	}
	return overrides, nil
}

func rewriteFile(path string, transform func([]byte) ([]byte, error)) error {
	// #nosec G304 -- every caller supplies a repository-owned generated path.
	input, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	output, err := transform(input)
	if err != nil {
		return fmt.Errorf("transform %s: %w", path, err)
	}
	if bytes.Equal(input, output) {
		return nil
	}
	// #nosec G306 -- generated source artifacts intentionally use repository file permissions.
	if err := os.WriteFile(path, output, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func applyGoHTTPStatus(input []byte, override statusOverride) ([]byte, error) {
	if override.status != http.StatusNoContent {
		output, err := applyGoHTTPServerBodyStatus(string(input), override)
		return []byte(output), err
	}
	output, err := applyGoHTTPServerStatus(string(input), override)
	if err != nil {
		return nil, err
	}
	output, err = applyGoHTTPClientStatus(output, override)
	if err != nil {
		return nil, err
	}
	return []byte(ensureGoImport(output, `httpbody "google.golang.org/genproto/googleapis/api/httpbody"`)), nil
}

func applyGoHTTPServerBodyStatus(input string, override statusOverride) (string, error) {
	functionPrefix := "func _" + override.serviceName + "_" + override.methodName
	start := strings.Index(input, functionPrefix)
	if start < 0 {
		return "", fmt.Errorf("handler not found for %s.%s", override.serviceName, override.methodName)
	}
	relativeEnd := strings.Index(input[start:], "\n}\n")
	if relativeEnd < 0 {
		return "", fmt.Errorf("unterminated handler for %s.%s", override.serviceName, override.methodName)
	}
	end := start + relativeEnd + len("\n}\n")
	segment := input[start:end]
	generatedResult := "\t\treturn ctx.Result(200, reply)"
	if !strings.Contains(segment, generatedResult) {
		return "", fmt.Errorf("200 response result not found for %s.%s", override.serviceName, override.methodName)
	}
	segment = strings.Replace(
		segment,
		generatedResult,
		"\t\treturn ctx.Result("+strconv.Itoa(override.status)+", reply)",
		1,
	)
	return input[:start] + segment + input[end:], nil
}

func applyGoHTTPServerStatus(input string, override statusOverride) (string, error) {
	output := input
	functionPrefix := "func _" + override.serviceName + "_" + override.methodName
	searchFrom := 0
	replacements := 0
	for {
		relativeStart := strings.Index(output[searchFrom:], functionPrefix)
		if relativeStart < 0 {
			break
		}
		start := searchFrom + relativeStart
		relativeEnd := strings.Index(output[start:], "\n}\n")
		if relativeEnd < 0 {
			return "", fmt.Errorf("unterminated handler for %s.%s", override.serviceName, override.methodName)
		}
		end := start + relativeEnd + len("\n}\n")
		segment := output[start:end]
		resultStart := strings.Index(segment, "\t\treply := out.(*")
		resultEndMarker := "\n\t\treturn ctx.Result(200, reply)"
		resultEnd := strings.Index(segment, resultEndMarker)
		if resultStart < 0 || resultEnd < resultStart {
			return "", fmt.Errorf("200 response block not found for %s.%s", override.serviceName, override.methodName)
		}
		resultEnd += len(resultEndMarker)
		replacement := "\t\t_ = out\n\t\tctx.Response().WriteHeader(" + strconv.Itoa(override.status) + ")\n\t\treturn nil"
		segment = segment[:resultStart] + replacement + segment[resultEnd:]
		output = output[:start] + segment + output[end:]
		searchFrom = start + len(segment)
		replacements++
	}
	if replacements == 0 {
		return "", fmt.Errorf("handler not found for %s.%s", override.serviceName, override.methodName)
	}
	return output, nil
}

func applyGoHTTPClientStatus(input string, override statusOverride) (string, error) {
	functionPrefix := "func (c *" + override.serviceName + "HTTPClientImpl) " + override.methodName + "("
	start := strings.Index(input, functionPrefix)
	if start < 0 {
		return "", fmt.Errorf("client method not found for %s.%s", override.serviceName, override.methodName)
	}
	relativeEnd := strings.Index(input[start:], "\n}\n")
	if relativeEnd < 0 {
		return "", fmt.Errorf("unterminated client method for %s.%s", override.serviceName, override.methodName)
	}
	end := start + relativeEnd + len("\n}\n")
	segment := input[start:end]
	const generatedReply = ", &out, opts...)"
	if !strings.Contains(segment, generatedReply) {
		return "", fmt.Errorf("client response target not found for %s.%s", override.serviceName, override.methodName)
	}
	segment = strings.Replace(segment, generatedReply, ", &httpbody.HttpBody{}, opts...)", 1)
	return input[:start] + segment + input[end:], nil
}

func ensureGoImport(input, importLine string) string {
	if strings.Contains(input, importLine) {
		return input
	}
	const importEnd = ")\n\n// This is a compile-time assertion"
	return strings.Replace(input, importEnd, "\t"+importLine+"\n"+importEnd, 1)
}

func rewriteGeneratedHTTPClients(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_http.pb.go") {
			return nil
		}
		return rewriteFile(path, func(input []byte) ([]byte, error) {
			output := applyGoHTTPClientPathVariables(input)
			return applyGoHTTPClientResponseBodies(output), nil
		})
	})
}

func applyGoHTTPClientPathVariables(input []byte) []byte {
	lines := strings.Split(string(input), "\n")
	for index, line := range lines {
		if strings.Contains(line, `pattern := "`) {
			lines[index] = normalizePathVariables(line)
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func applyGoHTTPClientResponseBodies(input []byte) []byte {
	output := string(input)
	searchFrom := 0
	for {
		relativeInvoke := strings.Index(output[searchFrom:], "c.cc.Invoke(")
		if relativeInvoke < 0 {
			return []byte(output)
		}
		invoke := searchFrom + relativeInvoke
		lineEnd := strings.IndexByte(output[invoke:], '\n')
		if lineEnd < 0 {
			lineEnd = len(output) - invoke
		}
		if strings.Contains(output[invoke:invoke+lineEnd], ", &out.") {
			methodStart := strings.LastIndex(output[:invoke], "func (c *")
			if methodStart >= 0 {
				segment := output[methodStart:invoke]
				const protoJSONAccept = `http.Accept("application/protojson")`
				accept := strings.LastIndex(segment, protoJSONAccept)
				if accept >= 0 {
					accept += methodStart
					output = output[:accept] + `http.Accept("application/json")` + output[accept+len(protoJSONAccept):]
				}
			}
		}
		searchFrom = invoke + lineEnd
	}
}

func normalizePathVariables(input string) string {
	var output strings.Builder
	for {
		start := strings.IndexByte(input, '{')
		if start < 0 {
			_, _ = output.WriteString(input)
			return output.String()
		}
		end := strings.IndexByte(input[start:], '}')
		if end < 0 {
			_, _ = output.WriteString(input)
			return output.String()
		}
		end += start
		_, _ = output.WriteString(input[:start+1])
		_, _ = output.WriteString(jsonNamePath(input[start+1 : end]))
		_ = output.WriteByte('}')
		input = input[end+1:]
	}
}

func jsonNamePath(input string) string {
	fieldPath, suffix, found := strings.Cut(input, "=")
	parts := strings.Split(fieldPath, ".")
	for index, part := range parts {
		parts[index] = snakeToLowerCamel(part)
	}
	result := strings.Join(parts, ".")
	if found {
		return result + "=" + suffix
	}
	return result
}

func snakeToLowerCamel(input string) string {
	var output strings.Builder
	upperNext := false
	for _, character := range input {
		if character == '_' {
			upperNext = true
			continue
		}
		if upperNext && character >= 'a' && character <= 'z' {
			character -= 'a' - 'A'
		}
		upperNext = false
		_, _ = output.WriteRune(character)
	}
	return output.String()
}

func applyOpenAPIStatuses(input []byte, overrides []statusOverride) ([]byte, error) {
	lines := strings.Split(string(input), "\n")
	for _, override := range overrides {
		var err error
		lines, err = applyOpenAPIOperationStatus(lines, override)
		if err != nil {
			return nil, err
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func applyOpenAPIResponseBodies(input []byte, overrides []responseBodyOverride) ([]byte, error) {
	lines := strings.Split(string(input), "\n")
	for _, override := range overrides {
		var err error
		lines, err = applyOpenAPIResponseBody(lines, override)
		if err != nil {
			return nil, err
		}
		lines, err = applyOpenAPIOptionalFields(lines, override)
		if err != nil {
			return nil, err
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func applyOpenAPIOptionalFields(lines []string, override responseBodyOverride) ([]string, error) {
	if len(override.optionalOpenAPIFields) == 0 {
		return lines, nil
	}
	schemaLine := findTrimmedLine(lines, override.schemaName+":", 0)
	if schemaLine < 0 {
		return nil, fmt.Errorf("OpenAPI component schema %s not found", override.schemaName)
	}
	schemaIndent := indentation(lines[schemaLine])
	schemaEnd := schemaLine + 1
	for schemaEnd < len(lines) && (strings.TrimSpace(lines[schemaEnd]) == "" || indentation(lines[schemaEnd]) > schemaIndent) {
		schemaEnd++
	}
	for _, fieldName := range override.optionalOpenAPIFields {
		fieldLine := findTrimmedLineBefore(lines, fieldName+":", schemaLine+1, schemaEnd)
		if fieldLine < 0 {
			return nil, fmt.Errorf("OpenAPI optional field %s.%s not found", override.schemaName, fieldName)
		}
		fieldIndent := indentation(lines[fieldLine])
		fieldEnd := fieldLine + 1
		for fieldEnd < schemaEnd && strings.TrimSpace(lines[fieldEnd]) != "" && indentation(lines[fieldEnd]) > fieldIndent {
			if strings.TrimSpace(lines[fieldEnd]) == "nullable: true" {
				fieldEnd = -1
				break
			}
			fieldEnd++
		}
		if fieldEnd == -1 {
			continue
		}
		lines = append(lines[:fieldEnd], append([]string{strings.Repeat(" ", fieldIndent+4) + "nullable: true"}, lines[fieldEnd:]...)...)
		schemaEnd++
	}
	return lines, nil
}

func applyOpenAPIResponseBody(lines []string, override responseBodyOverride) ([]string, error) {
	operationID := override.serviceName + "_" + override.methodName
	operationLine := findTrimmedLine(lines, "operationId: "+operationID, 0)
	if operationLine < 0 {
		return nil, fmt.Errorf("OpenAPI operation %s not found", operationID)
	}
	responsesLine := findTrimmedLine(lines, "responses:", operationLine+1)
	if responsesLine < 0 {
		return nil, fmt.Errorf("OpenAPI responses for %s not found", operationID)
	}
	responseIndent := indentation(lines[responsesLine]) + 4
	statusLine := findResponseStatusLine(lines, responsesLine+1, responseIndent, `"200":`)
	if statusLine < 0 {
		return nil, fmt.Errorf("OpenAPI 200 response for %s not found", operationID)
	}
	nextResponse := findNextResponseLine(lines, statusLine+1, responseIndent)
	schemaLine := findTrimmedLineBefore(lines, "schema:", statusLine+1, nextResponse)
	if schemaLine < 0 {
		return nil, fmt.Errorf("OpenAPI response schema for %s not found", operationID)
	}
	schemaIndent := indentation(lines[schemaLine])
	schemaEnd := schemaLine + 1
	for schemaEnd < nextResponse && (strings.TrimSpace(lines[schemaEnd]) == "" || indentation(lines[schemaEnd]) > schemaIndent) {
		schemaEnd++
	}
	prefix := strings.Repeat(" ", schemaIndent+4)
	replacement := []string{
		lines[schemaLine],
		prefix + "type: array",
		prefix + "items:",
		prefix + "    $ref: '#/components/schemas/" + override.schemaName + "'",
	}
	return append(lines[:schemaLine], append(replacement, lines[schemaEnd:]...)...), nil
}

func findTrimmedLineBefore(lines []string, target string, start, end int) int {
	for index := start; index < end && index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == target {
			return index
		}
	}
	return -1
}

func applyOpenAPIOperationStatus(lines []string, override statusOverride) ([]string, error) {
	operationID := override.serviceName + "_" + override.methodName
	operationLine := findTrimmedLine(lines, "operationId: "+operationID, 0)
	if operationLine < 0 {
		return nil, fmt.Errorf("OpenAPI operation %s not found", operationID)
	}
	responsesLine := findTrimmedLine(lines, "responses:", operationLine+1)
	if responsesLine < 0 {
		return nil, fmt.Errorf("OpenAPI responses for %s not found", operationID)
	}
	responseIndent := indentation(lines[responsesLine]) + 4
	statusLine := findResponseStatusLine(lines, responsesLine+1, responseIndent, "\"200\":")
	if statusLine < 0 {
		return nil, fmt.Errorf("OpenAPI 200 response for %s not found", operationID)
	}
	prefix := strings.Repeat(" ", responseIndent)
	if override.status != http.StatusNoContent {
		lines[statusLine] = prefix + "\"" + strconv.Itoa(override.status) + "\":"
		descriptionLine := statusLine + 1
		if descriptionLine < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[descriptionLine]), "description:") {
			lines[descriptionLine] = strings.Repeat(" ", responseIndent+4) + "description: " + http.StatusText(override.status)
		}
		return lines, nil
	}
	nextResponse := findNextResponseLine(lines, statusLine+1, responseIndent)
	replacement := []string{
		prefix + "\"" + strconv.Itoa(override.status) + "\":",
		prefix + "    description: No Content",
	}
	return append(lines[:statusLine], append(replacement, lines[nextResponse:]...)...), nil
}

func findResponseStatusLine(lines []string, start, responseIndent int, status string) int {
	for index := start; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if indentation(lines[index]) < responseIndent && trimmed != "" {
			break
		}
		if indentation(lines[index]) == responseIndent && trimmed == status {
			return index
		}
	}
	return -1
}

func findNextResponseLine(lines []string, start, responseIndent int) int {
	for index := start; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "" && indentation(lines[index]) <= responseIndent {
			return index
		}
	}
	return len(lines)
}

func findTrimmedLine(lines []string, target string, start int) int {
	for index := start; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == target {
			return index
		}
	}
	return -1
}

func indentation(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
