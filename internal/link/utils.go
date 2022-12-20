package link

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func applyPatch(workingDirectory string, patch io.Reader) error {
	cmd := exec.Command("git", "-C", workingDirectory, "apply")
	cmd.Stdin = patch
	return cmd.Run()
}

type overlayFile struct {
	Replace map[string]string `json:"Replace"`
}

func compileLinker(workingDirectory string, overlay map[string]string, outputLinkPath string) error {
	file, _ := json.Marshal(&overlayFile{Replace: overlay})
	overlayPath := filepath.Join(workingDirectory, "overlay.json")

	if err := os.WriteFile(overlayPath, file, os.ModePerm); err != nil {
		return err
	}

	out, err := exec.Command("go", "build", "-overlay", overlayPath, "-o", outputLinkPath, "cmd/link").CombinedOutput()
	if err != nil {
		return fmt.Errorf("compiler compile error: %v\n\n%s", err, string(out))
	}
	return nil
}

func existsFile(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func copyFile(src, target string) error {
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, os.ModeDir); err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	targetFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer targetFile.Close()
	_, err = io.Copy(targetFile, srcFile)
	return err
}
