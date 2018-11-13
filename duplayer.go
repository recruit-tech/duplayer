package duplayer

import (
	"archive/tar"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
)

const (
	opq            string = ".wh..wh..opq"
	wh             string = ".wh."
	humanizedWidth        = 7
)

type manifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

type Files struct {
	whFiles    map[string]int64
	opqDirs    map[string]int64
	filePaths  map[string]int64
	numOfFiles int64
}

type FilesInfo struct {
	totalSize  int64
	numOfFiles int64
	files      FileInfos
}

type FileInfo struct {
	filePath string
	fileSize int64
}
type FileInfos []FileInfo

func (a FileInfos) Len() int           { return len(a) }
func (a FileInfos) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a FileInfos) Less(i, j int) bool { return a[i].fileSize < a[j].fileSize }

type History struct {
	Created    time.Time `json:"created"`
	Author     string    `json:"author,omitempty"`
	CreatedBy  string    `json:"created_by,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	EmptyLayer bool      `json:"empty_layer,omitempty"`
}

type Image struct {
	History []History `json:"history,omitempty"`
}

type Layer struct {
	layerID string
	files   Files
	cmd     string
	size    int64
}

type Layers []*Layer
type LayersMap map[string]Layers

func Duplayer() error {

	tarPath := flag.String("f", "-", "layer.tar path")
	saveLimitSize := flag.Int("l", 10, "min save size for showing( KB )")
	maxFileNum := flag.Int("M", 10, "max num of duplicate filePath for showing")
	showFileSize := flag.Int("m", 10, "min size of duplicate filePath for showing")

	lineWidth := flag.Int("w", 100, "screen line width")

	verbose := flag.Bool("v", false, "show verbose")
	flag.Parse()
	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}
	rc, err := openStream(*tarPath)
	if err != nil {
		return err
	}
	layersMap, err := readLayers(rc)
	if err != nil {
		return err
	}
	if err := showDuplicate(layersMap, *lineWidth, *saveLimitSize, *maxFileNum, *showFileSize); err != nil {
		return err
	}
	return nil

}

func showDuplicate(layersMap LayersMap, lineWidth int, saveLimitSize int, maxFileNum int, showFileSize int) error {

	for repoTag, layers := range layersMap {
		fmt.Println(strings.Repeat("=", lineWidth))
		fmt.Printf("RepoTag[0] : %s\n", repoTag)

		for i, layer := range layers {
			fmt.Printf("[%d] %s \t %dfiles\t $ %s\n", i, humanizeBytes(layer.size), layer.files.numOfFiles, layer.cmd)
		}

		fmt.Println(strings.Repeat("=", lineWidth))

		fmt.Printf("\nif you merge [lower] and [upper] save num_of_files data_size (only show over %dKB save)\n\n", saveLimitSize)

		for i, _ := range layers {
			if i+1 < len(layers) {
				for j := i + 1; j < len(layers); j++ {
					lower := layers[i]
					upper := layers[j]
					dupInfo := upper.checkDuplicateFiles(lower)
					if dupInfo.totalSize > int64(saveLimitSize*1024) {
						fmt.Println(strings.Repeat("=", lineWidth))
						fmt.Printf("[%d] %s\n", i, lower.cmd)
						fmt.Printf("[%d] %s\n", j, upper.cmd)
						fmt.Printf("save : %d files (%s)\n", dupInfo.numOfFiles, humanizeBytes(dupInfo.totalSize))
						sort.Sort(sort.Reverse(dupInfo.files))
						for k := 0; k < len(dupInfo.files) && k < maxFileNum; k++ {
							if dupInfo.files[k].fileSize > int64(showFileSize*1024) {
								fmt.Printf("%s\t%s\n", humanizeBytes(dupInfo.files[k].fileSize), dupInfo.files[k].filePath)
							} else {
								break
							}
						}
					}
				}
			}
		}
		fmt.Println(strings.Repeat("=", lineWidth))
	}
	return nil
}

func (upper *Layer) checkDuplicateFiles(lower *Layer) FilesInfo {
	dupInfo := FilesInfo{0, 0, []FileInfo{}}
	for lowerFile, fileSize := range lower.files.filePaths {
		if upper.files.isDuplicate(lowerFile) {
			dupInfo.totalSize += fileSize
			dupInfo.numOfFiles++
			dupInfo.files = append(dupInfo.files, FileInfo{lowerFile, fileSize})
		}
	}
	return dupInfo
}

func (upper Files) isDuplicate(path string) bool {
	if _, ok := upper.filePaths[path]; ok == true {
		return true
	}
	if _, ok := upper.whFiles[path]; ok == true {
		return true
	}
	for p := filepath.Dir(path); p != "."; p = filepath.Dir(p) {
		if _, ok := upper.whFiles[p]; ok == true {
			return true
		}
		if _, ok := upper.opqDirs[p]; ok == true {
			return true
		}
	}
	return false
}

func readLayers(rc io.ReadCloser) (LayersMap, error) {
	defer rc.Close()
	archive := tar.NewReader(rc)
	var manifests []manifest
	filesInLayers := make(map[string]Files)
	imageMetas := make(map[string]Image)
	sizeMap := make(map[string]int64)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		switch {
		case header.Name == "manifest.json":
			if err := json.NewDecoder(archive).Decode(&manifests); err != nil {
				return nil, err
			}
		case strings.HasSuffix(header.Name, ".tar"):
			layerID := filepath.Base(filepath.Dir(header.Name))
			files, err := getFilesInLayer(archive)
			if err != nil {
				return nil, err
			}
			filesInLayers[layerID] = files
			sizeMap[layerID] = header.Size
		case strings.HasSuffix(header.Name, ".json"):
			var imageMeta Image
			if err := json.NewDecoder(archive).Decode(&imageMeta); err != nil {
				return nil, err
			}
			imageMetas[header.Name] = imageMeta
		default:
		}
	}
	layersMap, err := makeMetaData(manifests, filesInLayers, imageMetas, sizeMap)
	if err != nil {
		return nil, err
	}
	return layersMap, nil
}

func makeMetaData(manifests []manifest, files map[string]Files, imageMetas map[string]Image, sizeMap map[string]int64) (LayersMap, error) {
	layersMap := make(LayersMap)
	for _, manifest := range manifests {
		var layers Layers
		i := 0
		for _, history := range imageMetas[manifest.Config].History {
			if !history.EmptyLayer {
				layerID := filepath.Base(filepath.Dir(manifest.Layers[i]))
				layer := &Layer{layerID, files[layerID], history.CreatedBy, sizeMap[layerID]}
				layers = append(layers, layer)
				i++
			}
		}
		layersMap[manifest.RepoTags[0]] = layers
	}
	return layersMap, nil
}

func getFilesInLayer(image *tar.Reader) (Files, error) {
	archive := tar.NewReader(image)
	imgFile := Files{make(map[string]int64), make(map[string]int64), make(map[string]int64), 0}
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return imgFile, err
		}
		fullPath := filepath.Clean(header.Name)
		fileName := filepath.Base(fullPath)
		fileDir := filepath.Dir(fullPath)
		imgFile.numOfFiles++
		switch {
		case fileName == opq:
			imgFile.opqDirs[fileDir] = header.Size
		case filepath.HasPrefix(filepath.Base(header.Name), wh):
			whFile := filepath.Join(fileDir, strings.TrimPrefix(fileName, wh))
			imgFile.whFiles[whFile] = header.Size
		default:
			imgFile.filePaths[fullPath] = header.Size
		}
	}
	return imgFile, nil
}

func humanizeBytes(sz int64) string {
	return pad(humanize.Bytes(uint64(sz)), humanizedWidth)
}

func pad(s string, n int) string {
	return strings.Repeat(" ", n-len(s)) + s
}

func openStream(path string) (*os.File, error) {
	if path == "-" {
		return os.Stdin, nil
	} else {
		return os.Open(path)
	}
}
