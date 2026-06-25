package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type TaskRequestSummary struct {
	Version  int                  `json:"version"`
	Type     string               `json:"type"`
	Command  string               `json:"command"`
	Fields   map[string]any       `json:"fields,omitempty"`
	Payloads []TaskPayloadSummary `json:"payloads,omitempty"`
}

type TaskPayloadSummary struct {
	Path    string `json:"path"`
	Size    int    `json:"size"`
	SHA256  string `json:"sha256"`
	Omitted bool   `json:"omitted"`
}

func BuildTaskRequestSummary(spite *implantpb.Spite) *TaskRequestSummary {
	if spite == nil {
		return &TaskRequestSummary{Version: 1}
	}

	payloads := make([]TaskPayloadSummary, 0)
	fields := summarizeMessage(spite.ProtoReflect(), nil, &payloads)

	return &TaskRequestSummary{
		Version:  1,
		Type:     spite.GetName(),
		Command:  BuildTaskCommandSummary(spite),
		Fields:   fields,
		Payloads: payloads,
	}
}

func BuildTaskRequestSummaryJSON(spite *implantpb.Spite) (string, error) {
	data, err := json.Marshal(BuildTaskRequestSummary(spite))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func BuildTaskCommandSummary(spite *implantpb.Spite) string {
	if spite == nil {
		return ""
	}

	switch body := spite.GetBody().(type) {
	case *implantpb.Spite_ExecuteBinary:
		return summarizeExecuteBinary(body.ExecuteBinary, spite.GetName())
	case *implantpb.Spite_Request:
		req := body.Request
		if req == nil {
			return spite.GetName()
		}
		return summarizeRequest(req)
	case *implantpb.Spite_Common:
		common := body.Common
		if common == nil {
			return spite.GetName()
		}
		return common.GetName()
	case *implantpb.Spite_ExecRequest:
		return summarizeExecRequest(body.ExecRequest, spite.GetName())
	case *implantpb.Spite_UploadRequest:
		req := body.UploadRequest
		return fmt.Sprintf("upload %s -> %s", req.GetName(), req.GetTarget())
	case *implantpb.Spite_DownloadRequest:
		req := body.DownloadRequest
		return fmt.Sprintf("download %s", req.GetPath())
	case *implantpb.Spite_LoadModule:
		req := body.LoadModule
		return fmt.Sprintf("load_module %s", req.GetBundle())
	case *implantpb.Spite_LoadAddon:
		req := body.LoadAddon
		return fmt.Sprintf("load_addon %s", req.GetName())
	case *implantpb.Spite_CurlRequest:
		req := body.CurlRequest
		return strings.TrimSpace(fmt.Sprintf("curl %s %s", req.GetMethod(), req.GetUrl()))
	case *implantpb.Spite_PtyRequest:
		return summarizePtyRequest(body.PtyRequest, spite.GetName())
	default:
		return spite.GetName()
	}
}

func summarizeRequest(req *implantpb.Request) string {
	if req == nil {
		return ""
	}
	parts := []string{req.GetName()}
	if input := req.GetInput(); input != "" {
		parts = append(parts, input)
	}
	parts = append(parts, req.GetArgs()...)
	return joinCommandParts(parts)
}

func summarizeExecRequest(req *implantpb.ExecRequest, fallback string) string {
	if req == nil {
		return fallback
	}
	parts := []string{"exec"}
	if req.GetPath() != "" {
		parts = append(parts, req.GetPath())
	}
	if len(req.GetArgs()) > 0 {
		parts = append(parts, "--")
		parts = append(parts, req.GetArgs()...)
	}
	return joinCommandParts(parts)
}

func summarizeExecuteBinary(binary *implantpb.ExecuteBinary, fallback string) string {
	if binary == nil {
		return fallback
	}
	name := binary.GetName()
	if name == "" {
		name = binary.GetPath()
	}
	command := binary.GetType()
	if command == "" {
		command = fallback
	}
	parts := []string{command}
	if name != "" {
		parts = append(parts, name)
	}
	if len(binary.GetArgs()) > 0 {
		parts = append(parts, "--")
		parts = append(parts, binary.GetArgs()...)
	}
	return joinCommandParts(parts)
}

func summarizePtyRequest(req *implantpb.PtyRequest, fallback string) string {
	if req == nil {
		return fallback
	}
	parts := []string{"pty"}
	if req.GetType() != "" {
		parts = append(parts, req.GetType())
	}
	if req.GetShell() != "" {
		parts = append(parts, req.GetShell())
	}
	if req.GetInputText() != "" {
		parts = append(parts, req.GetInputText())
	}
	return joinCommandParts(parts)
}

func joinCommandParts(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		quoted = append(quoted, quoteCommandArg(part))
	}
	return strings.Join(quoted, " ")
}

func quoteCommandArg(arg string) string {
	if arg == "" {
		return arg
	}
	if strings.ContainsAny(arg, " \t\r\n\"'\\") {
		return strconv.Quote(arg)
	}
	return arg
}

func summarizeMessage(message protoreflect.Message, path []string, payloads *[]TaskPayloadSummary) map[string]any {
	fields := make(map[string]any)
	message.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		name := string(fd.Name())
		fieldPath := append(path, name)
		fields[name] = summarizeField(fd, value, fieldPath, payloads)
		return true
	})
	return fields
}

func summarizeField(fd protoreflect.FieldDescriptor, value protoreflect.Value, path []string, payloads *[]TaskPayloadSummary) any {
	if fd.IsMap() {
		return summarizeMap(fd, value.Map(), path, payloads)
	}
	if fd.IsList() {
		return summarizeList(fd, value.List(), path, payloads)
	}
	return summarizeValue(fd, value, path, payloads)
}

func summarizeMap(fd protoreflect.FieldDescriptor, values protoreflect.Map, path []string, payloads *[]TaskPayloadSummary) map[string]any {
	result := make(map[string]any)
	keys := make([]protoreflect.MapKey, 0, values.Len())
	values.Range(func(key protoreflect.MapKey, _ protoreflect.Value) bool {
		keys = append(keys, key)
		return true
	})
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})

	valueDescriptor := fd.MapValue()
	for _, key := range keys {
		keyString := fmt.Sprint(key.Interface())
		result[keyString] = summarizeValue(valueDescriptor, values.Get(key), append(path, keyString), payloads)
	}
	return result
}

func summarizeList(fd protoreflect.FieldDescriptor, values protoreflect.List, path []string, payloads *[]TaskPayloadSummary) []any {
	result := make([]any, 0, values.Len())
	for i := 0; i < values.Len(); i++ {
		result = append(result, summarizeValue(fd, values.Get(i), append(path, strconv.Itoa(i)), payloads))
	}
	return result
}

func summarizeValue(fd protoreflect.FieldDescriptor, value protoreflect.Value, path []string, payloads *[]TaskPayloadSummary) any {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return value.Bool()
	case protoreflect.EnumKind:
		enum := fd.Enum().Values().ByNumber(value.Enum())
		if enum == nil {
			return int32(value.Enum())
		}
		return string(enum.Name())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return int32(value.Int())
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return value.Int()
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return uint32(value.Uint())
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return value.Uint()
	case protoreflect.FloatKind:
		return float32(value.Float())
	case protoreflect.DoubleKind:
		return value.Float()
	case protoreflect.StringKind:
		return value.String()
	case protoreflect.BytesKind:
		summary := summarizeBytes(value.Bytes(), strings.Join(path, "."))
		*payloads = append(*payloads, summary)
		return summary
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return summarizeMessage(value.Message(), path, payloads)
	default:
		return fmt.Sprint(value.Interface())
	}
}

func summarizeBytes(data []byte, path string) TaskPayloadSummary {
	sum := sha256.Sum256(data)
	return TaskPayloadSummary{
		Path:    path,
		Size:    len(data),
		SHA256:  hex.EncodeToString(sum[:]),
		Omitted: true,
	}
}
