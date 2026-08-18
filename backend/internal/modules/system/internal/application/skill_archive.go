package application

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/hvritual/workspace/internal/modules/system/contract"
	"golang.org/x/text/unicode/norm"
)

const (
	SkillImportMaxCompressedBytes = 10 << 20
	SkillImportMaxTotalBytes      = 50 << 20
	SkillImportMaxFiles           = 500
	SkillImportMaxFileBytes       = 5 << 20
	SkillImportMaxPathDepth       = 16
	SkillImportMaxPathBytes       = 512
)

func ValidateSkillArchive(data []byte) (contract.SkillImportPreview, error) {
	if len(data) > SkillImportMaxCompressedBytes {
		return contract.SkillImportPreview{}, errors.New("Skill archive exceeds compressed size limit")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return contract.SkillImportPreview{}, errors.New("invalid .skill/.zip archive")
	}
	root, err := skillArchiveRoot(reader.File)
	if err != nil {
		return contract.SkillImportPreview{}, err
	}
	preview := contract.SkillImportPreview{Files: make([]contract.SkillFileBody, 0, len(reader.File)), Warnings: []string{}}
	seen := make(map[string]struct{}, len(reader.File))
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		canonical, err := canonicalArchivePath(entry.Name)
		if err != nil {
			return contract.SkillImportPreview{}, err
		}
		if root != "" {
			if !strings.HasPrefix(canonical, root) {
				return contract.SkillImportPreview{}, fmt.Errorf("archive entry %q is outside the Skill root", entry.Name)
			}
			canonical = strings.TrimPrefix(canonical, root)
		}
		if canonical == "" {
			return contract.SkillImportPreview{}, errors.New("empty canonical Skill path")
		}
		if err := validateCanonicalSkillPath(canonical); err != nil {
			return contract.SkillImportPreview{}, err
		}
		if !entry.Mode().IsRegular() {
			return contract.SkillImportPreview{}, fmt.Errorf("Skill entry %q is not a regular file", canonical)
		}
		key := norm.NFC.String(canonical)
		if _, exists := seen[key]; exists {
			return contract.SkillImportPreview{}, fmt.Errorf("duplicate canonical Skill path %q", key)
		}
		seen[key] = struct{}{}
		if len(seen) > SkillImportMaxFiles {
			return contract.SkillImportPreview{}, errors.New("Skill archive exceeds file count limit")
		}
		body, err := readBoundedSkillFile(entry)
		if err != nil {
			return contract.SkillImportPreview{}, err
		}
		preview.TotalBytes += int64(len(body))
		if preview.TotalBytes > SkillImportMaxTotalBytes {
			return contract.SkillImportPreview{}, errors.New("Skill archive exceeds decompressed size limit")
		}
		mediaType, err := allowedSkillMediaType(key, body)
		if err != nil {
			return contract.SkillImportPreview{}, err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(body))
		preview.Files = append(preview.Files, contract.SkillFileBody{Path: key, MediaType: mediaType, Content: body, Checksum: checksum, SizeBytes: int64(len(body))})
	}
	sort.Slice(preview.Files, func(i, j int) bool { return preview.Files[i].Path < preview.Files[j].Path })
	if len(preview.Files) == 0 || preview.Files[0].Path != "SKILL.md" {
		for _, file := range preview.Files {
			if file.Path == "SKILL.md" {
				goto foundSkill
			}
		}
		return contract.SkillImportPreview{}, errors.New("archive does not contain SKILL.md")
	}
foundSkill:
	for _, file := range preview.Files {
		if file.Path == "SKILL.md" {
			preview.Name, preview.Description = parseSkillFrontmatter(file.Content)
			break
		}
	}
	if preview.Name == "" {
		return contract.SkillImportPreview{}, errors.New("SKILL.md does not declare a name")
	}
	hash := sha256.New()
	for _, file := range preview.Files {
		_, _ = hash.Write([]byte(file.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(file.Checksum))
		_, _ = hash.Write([]byte{0})
	}
	preview.Checksum = fmt.Sprintf("%x", hash.Sum(nil))
	return preview, nil
}

func ValidateSkillFile(pathValue string, body []byte) (contract.SkillFileBody, error) {
	canonical, err := canonicalArchivePath(pathValue)
	if err != nil {
		return contract.SkillFileBody{}, err
	}
	if err := validateCanonicalSkillPath(canonical); err != nil {
		return contract.SkillFileBody{}, err
	}
	if len(body) > SkillImportMaxFileBytes {
		return contract.SkillFileBody{}, errors.New("Skill file exceeds individual size limit")
	}
	mediaType, err := allowedSkillMediaType(canonical, body)
	if err != nil {
		return contract.SkillFileBody{}, err
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(body))
	return contract.SkillFileBody{Path: canonical, MediaType: mediaType, Content: body, Checksum: checksum, SizeBytes: int64(len(body))}, nil
}

func skillArchiveRoot(entries []*zip.File) (string, error) {
	root := ""
	found := false
	for _, entry := range entries {
		canonical, err := canonicalArchivePath(entry.Name)
		if err != nil {
			return "", err
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if strings.EqualFold(path.Base(canonical), "SKILL.md") {
			candidate := strings.TrimSuffix(canonical, path.Base(canonical))
			if !found || strings.Count(candidate, "/") < strings.Count(root, "/") {
				root, found = candidate, true
			}
		}
	}
	if !found {
		return "", errors.New("archive does not contain SKILL.md")
	}
	return root, nil
}

func canonicalArchivePath(value string) (string, error) {
	if !utf8.ValidString(value) || value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsRune(value, 0) {
		return "", fmt.Errorf("invalid Skill path %q", value)
	}
	if len(value) >= 2 && value[1] == ':' {
		return "", fmt.Errorf("invalid drive Skill path %q", value)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("invalid Skill path segment in %q", value)
		}
		for _, character := range segment {
			if character < 0x20 || character == 0x7f {
				return "", fmt.Errorf("control character in Skill path %q", value)
			}
		}
	}
	return norm.NFC.String(value), nil
}

func validateCanonicalSkillPath(value string) error {
	if len([]byte(value)) > SkillImportMaxPathBytes {
		return fmt.Errorf("Skill path %q exceeds length limit", value)
	}
	if len(strings.Split(value, "/")) > SkillImportMaxPathDepth {
		return fmt.Errorf("Skill path %q exceeds depth limit", value)
	}
	return nil
}

func readBoundedSkillFile(entry *zip.File) ([]byte, error) {
	if entry.UncompressedSize64 > SkillImportMaxFileBytes {
		return nil, fmt.Errorf("Skill file %q exceeds individual size limit", entry.Name)
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("open Skill file %q: %w", entry.Name, err)
	}
	defer reader.Close()
	body, err := io.ReadAll(io.LimitReader(reader, SkillImportMaxFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Skill file %q: %w", entry.Name, err)
	}
	if len(body) > SkillImportMaxFileBytes {
		return nil, fmt.Errorf("Skill file %q exceeds individual size limit", entry.Name)
	}
	return body, nil
}

func allowedSkillMediaType(filename string, body []byte) (string, error) {
	lower := strings.ToLower(filename)
	for _, suffix := range []string{".zip", ".skill", ".tar", ".gz", ".tgz", ".7z", ".rar", ".exe", ".dll", ".so", ".dylib", ".msi", ".dmg", ".iso"} {
		if strings.HasSuffix(lower, suffix) {
			return "", fmt.Errorf("forbidden Skill file type %q", filename)
		}
	}
	if forbiddenSkillBinary(body) {
		return "", fmt.Errorf("forbidden Skill file content %q", filename)
	}
	if safeSkillText(body) {
		if value := mime.TypeByExtension(path.Ext(lower)); value != "" {
			return strings.Split(value, ";")[0], nil
		}
		return "text/plain", nil
	}
	detected := http.DetectContentType(body)
	if strings.HasPrefix(detected, "image/png") || strings.HasPrefix(detected, "image/jpeg") || strings.HasPrefix(detected, "image/gif") || strings.HasPrefix(detected, "image/webp") {
		return strings.Split(detected, ";")[0], nil
	}
	return "", fmt.Errorf("forbidden binary Skill file %q", filename)
}

func forbiddenSkillBinary(body []byte) bool {
	for _, signature := range [][]byte{
		[]byte("PK\x03\x04"), []byte("PK\x05\x06"), []byte("PK\x07\x08"),
		{0x1f, 0x8b}, {0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c}, []byte("Rar!\x1a\x07"),
		[]byte("MZ"), {0x7f, 'E', 'L', 'F'}, []byte("%PDF-"),
		[]byte("!<arch>\n"),
		{0xca, 0xfe, 0xba, 0xbe}, {0xfe, 0xed, 0xfa, 0xce}, {0xfe, 0xed, 0xfa, 0xcf},
		{0xce, 0xfa, 0xed, 0xfe}, {0xcf, 0xfa, 0xed, 0xfe}, {0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1},
	} {
		if bytes.HasPrefix(body, signature) {
			return true
		}
	}
	return len(body) >= 262 && bytes.Equal(body[257:262], []byte("ustar"))
}

func safeSkillText(body []byte) bool {
	if !utf8.Valid(body) {
		return false
	}
	for _, character := range string(body) {
		if character == 0 || (character < 0x20 && character != '\n' && character != '\r' && character != '\t') || character == 0x7f {
			return false
		}
	}
	return true
}

func parseSkillFrontmatter(body []byte) (string, string) {
	lines := strings.Split(string(body), "\n")
	inFrontmatter := len(lines) > 0 && strings.TrimSpace(lines[0]) == "---"
	name, description := "", ""
	if inFrontmatter {
		for _, line := range lines[1:] {
			trimmed := strings.TrimSpace(line)
			if trimmed == "---" {
				break
			}
			key, value, ok := strings.Cut(trimmed, ":")
			if !ok {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "name":
				name = strings.Trim(strings.TrimSpace(value), `"'`)
			case "description":
				description = strings.Trim(strings.TrimSpace(value), `"'`)
			}
		}
	}
	if name == "" {
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "# ") {
				name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "# "))
				break
			}
		}
	}
	return name, description
}
