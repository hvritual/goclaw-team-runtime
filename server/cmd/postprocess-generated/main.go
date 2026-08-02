// Command postprocess-generated normalizes Proto-declared HTTP behavior that
// the pinned upstream generators do not yet preserve consistently. It runs as
// the final deterministic step of `make generate`.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	annotationsv1 "github.com/multica-ai/multica/server/gen/go/annotations/v1"
	_ "github.com/multica-ai/multica/server/gen/go/auth/v1"
	_ "github.com/multica-ai/multica/server/gen/go/space/v1"
	_ "github.com/multica-ai/multica/server/gen/go/system/v1"
	_ "github.com/multica-ai/multica/server/gen/go/workspace/v1"
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
	if err := rewriteGeneratedHTTPClientPaths(filepath.Join("gen", "go")); err != nil {
		fatal(err)
	}
	openAPIPath := filepath.Join("gen", "openapi", "openapi.yaml")
	if err := rewriteFile(openAPIPath, func(input []byte) ([]byte, error) {
		return applyOpenAPIStatuses(input, overrides)
	}); err != nil {
		fatal(err)
	}
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

func rewriteGeneratedHTTPClientPaths(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_http.pb.go") {
			return nil
		}
		return rewriteFile(path, func(input []byte) ([]byte, error) {
			return applyGoHTTPClientPathVariables(input), nil
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
	nextResponse := findNextResponseLine(lines, statusLine+1, responseIndent)
	prefix := strings.Repeat(" ", responseIndent)
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
