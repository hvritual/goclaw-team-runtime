package application

import (
	"archive/zip"
	"bytes"
	"os"
	"strings"
	"testing"
)

type archiveEntry struct {
	name string
	body []byte
	mode os.FileMode
}

func TestValidateSkillArchiveNormalizesAndChecksumsCompleteManifest(t *testing.T) {
	data := buildSkillArchive(t,
		archiveEntry{name: "demo/SKILL.md", body: []byte("---\nname: Demo Skill\ndescription: Safe demo\n---\n# Demo")},
		archiveEntry{name: "demo/scripts/run.py", body: []byte("print('ok')\n")},
	)
	preview, err := ValidateSkillArchive(data)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Name != "Demo Skill" || preview.Description != "Safe demo" || len(preview.Files) != 2 || preview.Files[0].Path != "SKILL.md" || preview.Files[1].Path != "scripts/run.py" {
		t.Fatalf("preview = %#v", preview)
	}
	if preview.Checksum == "" || preview.Files[0].Checksum == "" || preview.TotalBytes != int64(len(preview.Files[0].Content)+len(preview.Files[1].Content)) {
		t.Fatalf("checksums/size = %#v", preview)
	}
}

func TestValidateSkillArchiveRejectsAdversarialEntriesAndLimits(t *testing.T) {
	deep := strings.Repeat("a/", 17) + "file.txt"
	duplicateA := "e\u0301.txt"
	duplicateB := "é.txt"
	for _, test := range []struct {
		name    string
		entries []archiveEntry
	}{
		{name: "missing SKILL md", entries: []archiveEntry{{name: "readme.md", body: []byte("no")}}},
		{name: "parent traversal", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "../escape", body: []byte("bad")}}},
		{name: "absolute", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "/escape", body: []byte("bad")}}},
		{name: "drive", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "C:/escape", body: []byte("bad")}}},
		{name: "backslash", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: `dir\escape`, body: []byte("bad")}}},
		{name: "symlink", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "link", body: []byte("target"), mode: os.ModeSymlink | 0o777}}},
		{name: "nested archive", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "nested.zip", body: []byte("PK\x03\x04bad")}}},
		{name: "renamed empty zip", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "notes.txt", body: []byte("PK\x05\x06\x00\x00")}}},
		{name: "renamed gzip", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "notes.txt", body: []byte{0x1f, 0x8b, 0x08, 0x00}}}},
		{name: "renamed pdf", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "notes.txt", body: []byte("%PDF-1.7")}}},
		{name: "renamed ar", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "notes.txt", body: []byte("!<arch>\nmember")}}},
		{name: "renamed tar", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "notes.txt", body: tarHeader()}}},
		{name: "nul text", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "notes.txt", body: []byte("safe\x00hidden")}}},
		{name: "duplicate canonical path", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: duplicateA, body: []byte("a")}, {name: duplicateB, body: []byte("b")}}},
		{name: "path depth", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: deep, body: []byte("bad")}}},
		{name: "individual size", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "large.txt", body: bytes.Repeat([]byte("x"), SkillImportMaxFileBytes+1)}}},
		{name: "binary executable", entries: []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}, {name: "evil.exe", body: []byte("MZ\x90\x00")}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ValidateSkillArchive(buildSkillArchive(t, test.entries...)); err == nil {
				t.Fatal("ValidateSkillArchive() error = nil")
			}
		})
	}
}

func TestValidateSkillFileRejectsInvalidUTF8Path(t *testing.T) {
	if _, err := ValidateSkillFile(string([]byte{'n', 'o', 't', 'e', '-', 0xff, '.', 't', 'x', 't'}), []byte("safe")); err == nil {
		t.Fatal("invalid UTF-8 path error = nil")
	}
}

func tarHeader() []byte {
	body := make([]byte, 512)
	copy(body[257:], []byte("ustar\x0000"))
	return body
}

func TestValidateSkillArchiveRejectsCompressedRequestAndFileCountLimits(t *testing.T) {
	if _, err := ValidateSkillArchive(make([]byte, SkillImportMaxCompressedBytes+1)); err == nil {
		t.Fatal("oversized compressed request error = nil")
	}
	entries := []archiveEntry{{name: "SKILL.md", body: []byte("# Safe")}}
	for i := 0; i < SkillImportMaxFiles; i++ {
		entries = append(entries, archiveEntry{name: strings.Repeat("a", i%20+1) + string(rune('A'+i%26)) + "-" + strings.Repeat("x", i/26) + ".txt", body: []byte("x")})
	}
	if _, err := ValidateSkillArchive(buildSkillArchive(t, entries...)); err == nil {
		t.Fatal("file-count error = nil")
	}
}

func buildSkillArchive(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.mode != 0 {
			header.SetMode(entry.mode)
		}
		file, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}
