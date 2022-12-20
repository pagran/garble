package link

import (
	"bufio"
	"crypto/sha1"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var (
	baseSubdir = filepath.Join("src", "cmd")

	//go:embed patches/*.patch
	linkPatches          embed.FS
	patchesVersion       string
	patchesModifiedFiles []string
)

func init() {
	tmpVersion, tmpModifiedFiles, err := getPatchesVersionAndModifiedFiles()
	if err != nil {
		panic(err)
	}
	patchesVersion = tmpVersion
	patchesModifiedFiles = tmpModifiedFiles
}

func PatchesVersion() string {
	return patchesVersion
}
func walkPatches(walkFunc fs.WalkDirFunc) error {
	return fs.WalkDir(linkPatches, "patches", walkFunc)
}

func getPatchesVersionAndModifiedFiles() (string, []string, error) {
	hash := sha1.New()
	var mod []string
	err := walkPatches(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		hash.Write([]byte(path))

		f, err := linkPatches.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			hash.Write(scanner.Bytes())

			text := scanner.Text()
			if !strings.HasPrefix(text, "Index:") {
				continue
			}

			fields := strings.Fields(text)
			if len(fields) != 2 {
				continue
			}
			mod = append(mod, fields[1])
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	return hex.EncodeToString(hash.Sum(nil)), mod, nil
}

func applyPatches(workingDirectory string) error {
	return walkPatches(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := linkPatches.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if err := applyPatch(workingDirectory, f); err != nil {
			return fmt.Errorf("apply patch %s failed: %v", path, err)
		}
		return nil
	})
}

func copyLinkerSrc(srcDir, workingDirectory string) (map[string]string, error) {
	overlay := make(map[string]string)
	for _, name := range patchesModifiedFiles {
		src := filepath.Join(srcDir, name)
		target := filepath.Join(workingDirectory, name)
		overlay[src] = target

		if err := copyFile(src, target); err != nil {
			return nil, err
		}
	}
	return overlay, nil
}

func PrepareModifiedLinker(goRoot, tempDirectory, outputLinkPath string) error {
	if existsFile(outputLinkPath) {
		return nil
	}

	srcDir := filepath.Join(goRoot, baseSubdir)
	workingDirectory := filepath.Join(tempDirectory, "link")

	overlay, err := copyLinkerSrc(srcDir, workingDirectory)
	if err != nil {
		return err
	}
	if err := applyPatches(workingDirectory); err != nil {
		return err
	}

	return compileLinker(workingDirectory, overlay, outputLinkPath)
}
