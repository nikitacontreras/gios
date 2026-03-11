package deploy

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InternalDebCreator implements a pure-Go debian packager to avoid dependency on 'dpkg-deb'
func InternalDebCreator(stageDir, outputFile string) error {
	debFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer debFile.Close()

	// 1. Write debian-binary (ar header + "2.0\n")
	// Format: magic + file1 + file2 + file3
	debFile.WriteString("!<arch>\n")

	writeArFile := func(name string, content []byte) {
		header := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`", 
			name, time.Now().Unix(), 0, 0, 0644, len(content))
		debFile.WriteString(header)
		if len(header)%2 != 0 { debFile.WriteString("\n") } // Alignment fixer if needed but ar spec is fixed 60
		debFile.Write(content)
		if len(content)%2 != 0 {
			debFile.WriteString("\n")
		}
	}

	writeArFile("debian-binary", []byte("2.0\n"))

	// 2. Create control.tar.gz
	controlBuf := new(bytes.Buffer)
	gw := gzip.NewWriter(controlBuf)
	tw := tar.NewWriter(gw)
	
	controlPath := filepath.Join(stageDir, "DEBIAN", "control")
	controlData, _ := os.ReadFile(controlPath)
	
	hdr := &tar.Header{
		Name: "control",
		Mode: 0644,
		Size: int64(len(controlData)),
	}
	tw.WriteHeader(hdr)
	tw.Write(controlData)
	tw.Close()
	gw.Close()
	writeArFile("control.tar.gz", controlBuf.Bytes())

	// 3. Create data.tar.gz
	dataBuf := new(bytes.Buffer)
	dgw := gzip.NewWriter(dataBuf)
	dtw := tar.NewWriter(dgw)

	filepath.Walk(stageDir, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, "DEBIAN") { return nil }
		if path == stageDir { return nil }

		relPath, _ := filepath.Rel(stageDir, path)
		relPath = filepath.ToSlash(relPath) // Ensure unix slashes in deb

		header, _ := tar.FileInfoHeader(info, "")
		header.Name = "./" + relPath
		
		if info.IsDir() {
			header.Name += "/"
		}

		dtw.WriteHeader(header)
		if !info.IsDir() {
			f, _ := os.Open(path)
			io.Copy(dtw, f)
			f.Close()
		}
		return nil
	})
	dtw.Close()
	dgw.Close()
	writeArFile("data.tar.gz", dataBuf.Bytes())

	return nil
}
