package comparer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	pathSep = string(os.PathSeparator)
)

var (
	// DocType is required If you want to use the package, so don't
	// forget to instantiate It before using the Compare function
	DocType string
	// Docs , Lines and InNeed are `metrics` of how the program is running
	Docs   int
	Lines  int
	InNeed int
)

// Compare is the function that takes two comparable paths to
// directories and writes It's differences into the translation's
// directories files
func Compare(original, translation string) (err error) {
	originalDir, err := ReadDir(original)
	if err != nil {
		return
	}
	for _, f := range originalDir {
		if f.IsDir() {
			err = checkTransDirExists(f.Name(), translation)
			if err != nil {
				return
			}
			err = Compare(filepath.Join(original, f.Name()), filepath.Join(translation, f.Name()))
			if err != nil {
				return
			}
		} else {
			Docs += 2
			err = readFiles(filepath.Join(original, f.Name()), filepath.Join(translation, f.Name()))
			if err != nil {
				return
			}
		}
	}
	return
}

func ReadDir(path string) (file []os.FileInfo, err error) {
	err = os.Chdir(path)
	if err != nil {
		return
	}
	fi, err := os.Open(path)
	if err != nil {
		return
	}
	defer fi.Close()
	file, err = fi.Readdir(0)
	if err != nil {
		return
	}
	return
}

func readFiles(originalFile, translationFile string) (err error) {
	if (len(originalFile) == 0) || (len(translationFile) == 0) {
		return errEmptyFileOrPathName
	}
	err = os.Chdir(filepath.Dir(originalFile))
	if err != nil {
		return
	}
	fName := strings.Split(originalFile, pathSep)
	fileName := fName[len(fName)-1]
	orgTags, err := readFile(fileName, filepath.Dir(originalFile))
	if err != nil {
		return
	}
	fName = strings.Split(translationFile, pathSep)
	fileName = fName[len(fName)-1]
	trltTags, err := readFile(fileName, filepath.Dir(translationFile))
	if err != nil {
		err = os.Chdir(filepath.Dir(translationFile))
		if err != nil {
			return
		}
		file, errCreate := os.Create(fileName)
		if errCreate != nil {
			return errCreate
		}
		defer file.Close()
	}
	missingTags := findMissing(orgTags, trltTags)
	if missingTags == nil {
		return
	}
	outdatedTags := findMissing(trltTags, orgTags)
	err = writeToFileMissingTags(translationFile, outdatedTags, true)
	if err != nil {
		return
	}
	err = writeToFileMissingTags(translationFile, missingTags, false)
	if err != nil {
		return
	}
	return
}

func writeToFileMissingTags(translationFilePath string, missingTags map[string]string, outdated bool) (err error) {
	f, err := os.OpenFile(translationFilePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	for missingKey, missingValue := range missingTags {
		if len(missingKey) == 0 {
			continue
		}
		if string(missingKey[1]) == pathSep {
			continue
		}
		InNeed++
		if isCommentaryOrDocType(missingKey) {
			_, err = f.WriteString(fmt.Sprintf("\n%s", missingKey))
			if err != nil {
				return
			}
			continue
		}
		if outdated {
			_, err = f.WriteString(fmt.Sprintf("\n[OUTDATED]%s", missingKey))
			if err != nil {
				return
			}
			continue
		}
		_, err = f.WriteString(fmt.Sprintf("\n%s%s</%s", missingKey, missingValue, missingKey[1:]))
		if err != nil {
			return
		}
	}
	return
}

func isCommentaryOrDocType(key string) bool {
	if (hasSubstring(key, "<!-")) || (hasSubstring(key, "<--")) || hasSubstring(key, "<?"+DocType) {
		return true
	}
	return false
}

func hasSubstring(str, s string) bool {
	return strings.Contains(s, str)
}

func readFile(fileName, filePath string) (map[string]string, error) {
	if (len(fileName) == 0) || (len(filePath) == 0) {
		return nil, errEmptyFileOrPathName
	}
	splittedFileName := strings.Split(fileName, ".")
	if splittedFileName[len(splittedFileName)-1] != DocType {
		return nil, nil
	}
	inFile, err := os.Open(filepath.Join(filePath, fileName))
	if err != nil {
		return nil, err
	}
	defer inFile.Close()
	tags := map[string]string{}
	scanner := bufio.NewScanner(inFile)
	for scanner.Scan() {
		Lines++
		line := scanner.Text()
		indexStart := strings.Index(line, "<")
		indexEnd := strings.Index(line, ">")
		if (len(line) == 0) || indexStart < 0 || indexEnd < 0 {
			continue
		}
		tag := line[indexStart : indexEnd+1]
		if string(tag[0]) == pathSep {
			continue
		}
		markers := strings.Split(tag, " ")
		tag = markers[0]
		valEnd := strings.LastIndex(line, "<")
		if valEnd < indexEnd {
			continue
		}
		translationValue := line[indexEnd+1 : valEnd]
		if (indexStart != -1) && (indexEnd != -1) {
			tags[tag] = translationValue
		}
	}
	return tags, nil
}

func findMissing(original, translation map[string]string) map[string]string {
	missing := make(map[string]string)
	if reflect.DeepEqual(original, translation) {
		return nil
	}
	for k, v := range original {
		if _, ok := translation[k]; !ok {
			missing[k] = v
		}
	}
	return missing
}

// TODO: refactor
// this should return a bool and an error,
// this way the upper func can handle dir creation
func checkTransDirExists(dir, translation string) (err error) {
	splitDir := strings.Split(dir, pathSep)
	dir = filepath.Join(translation, splitDir[len(splitDir)-1])
	_, err = os.Open(dir)
	if err != nil {
		splitedDirectory := strings.Split(dir, pathSep)
		parentDirFromSplit := dir[:len(dir)-len(splitedDirectory[len(splitedDirectory)-1])-1]
		os.Chdir(parentDirFromSplit)
		err = os.Mkdir(splitedDirectory[len(splitedDirectory)-1], 0700)
		if err != nil {
			return
		}
	}
	return
}
