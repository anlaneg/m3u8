package dl

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/anlaneg/m3u8/parse"
	"github.com/anlaneg/m3u8/tool"
	"strings"
)

const (
	tsExt               = ".ts"
	tsFolderName        = "ts"
	mergeTSFilename     = "main.ts"
	finishStateFileName = ".finished"
	tsTempFileSuffix    = "_tmp"
	progressWidth       = 40
)

type FileSlice struct {
	segId int
	tries int
}

type Downloader struct {
	lock     sync.Mutex
	queue    []*FileSlice
	folder   string
	tsFolder string
	finish   int32
	segLen   int

	result      *parse.Result
	fileName    string
	finishState *FinishState
}

// NewTask returns a Task instance
func NewTask(output string, url string) (*Downloader, error) {
	/*请求url,并获得result*/
	result, err := parse.FromURL(url)
	if err != nil {
		return nil, err
	}
	var folder string
	// If no output folder specified, use current directory
	if output == "" {
		/*默认使用当前目录*/
		current, err := tool.CurrentDir()
		if err != nil {
			return nil, err
		}
		folder = filepath.Join(current, output)
	} else {
		/*使用用户指定的目录*/
		folder = output
	}

	/*创建folder*/
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create storage folder failed: %s", err.Error())
	}
	/*创建ts folder*/
	tsFolder := filepath.Join(folder, tsFolderName)
	if err := os.MkdirAll(tsFolder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create ts folder '[%s]' failed: %s", tsFolder, err.Error())
	}

	/*构造downloader*/
	d := &Downloader{
		folder:      folder,
		tsFolder:    tsFolder,
		result:      result,
		finishState: nil,
	}

	/*加载finish状态*/
	state, err := d.loadFinishState()
	if err != nil {
		return nil, fmt.Errorf("load finish state '[%s]' failed: %s", filepath.Join(tsFolder, finishStateFileName), err.Error())
	}
	d.finishState = state

	/*指明总分片数*/
	d.segLen = len(result.M3u8.Segments)
	/*为各分片指定job id，创建job 队列*/
	d.queue = genSlice(d.segLen)
	d.fileName = GenFileName(url)
	return d, nil
}

// Start runs downloader
func (d *Downloader) Start(concurrency int, continueFlag bool, maxTries int) error {
	var wg sync.WaitGroup
	// struct{} zero size
	limitChan := make(chan struct{}, concurrency)
	for {
		/*取等执行job*/
		slice, end, err := d.next()
		if err != nil {
			if end {
				break
			}
			continue
		}
		wg.Add(1)
		go func(idx int, tries int) {
			defer wg.Done()
			/*针对idx号job执行download*/
			if err := d.proxyDownload(idx, continueFlag); err != nil {
				/*download时出错，将job扔回*/
				tries = tries + 1
				if maxTries <= 0 || tries < maxTries {
					// Back into the queue, retry request
					fmt.Printf("[failed] %s\n", err.Error())
					if err := d.back(idx,tries); err != nil {
						fmt.Printf(err.Error())
					}
				} else {
					fmt.Printf("[failed & giveup] %s\n", err.Error())
				}
			}
			<-limitChan
		}(slice.segId,slice.tries)
		limitChan <- struct{}{}
	}
	wg.Wait()
	/*任务完成，执行merge*/
	if err := d.merge(); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) proxyDownload(segIndex int, continueFlag bool) error {
	//tsFilename := tsFilename(segIndex)
	tsUrl := d.tsURL(segIndex)
	sign := "c"
	/*检查idx是否之前已完成下载*/
	finish := d.isFinished(segIndex)
	if !continueFlag || !finish {
		if err := d.download(segIndex); err != nil {
			return err
		}
	}

	/*增加完成的job*/
	atomic.AddInt32(&d.finish, 1)
	if !finish {
		err := d.updateFinishState(segIndex)
		if err != nil {
			return err
		}
		sign = "n"
	}
	//tool.DrawProgressBar("Downloading", float32(d.finish)/float32(d.segLen), progressWidth)
	/*显示进度*/
	fmt.Printf("[download(%s) %6.2f%%] %s\n", sign, float32(d.finish)/float32(d.segLen)*100, tsUrl)
	return nil
}

func (d *Downloader) loadFinishState() (*FinishState, error) {
	return LoadFinishState(filepath.Join(d.tsFolder, finishStateFileName))
}

func (d *Downloader) isFinished(segIndex int) bool {
	return d.finishState.isFinished(segIndex)
}

func (d *Downloader) updateFinishState(segIndex int) error {
	return d.finishState.updateFinishState(segIndex, filepath.Join(d.tsFolder, finishStateFileName))
}

/*执行segIndex号块的下载*/
func (d *Downloader) download(segIndex int) error {
	tsFilename := tsFilename(segIndex)
	tsUrl := d.tsURL(segIndex)
	/*请求tsurl*/
	b, e := tool.Get(tsUrl)
	if e != nil {
		return fmt.Errorf("request %s, %s", tsUrl, e.Error())
	}
	//noinspection GoUnhandledErrorResult
	defer b.Close()
	fPath := filepath.Join(d.tsFolder, tsFilename)
	fTemp := fPath + tsTempFileSuffix
	/*创建临时文件*/
	f, err := os.Create(fTemp)
	if err != nil {
		return fmt.Errorf("create file: %s, %s", tsFilename, err.Error())
	}

	/*拿到tsUrl对应内容*/
	bytes, err := ioutil.ReadAll(b)
	if err != nil {
		return fmt.Errorf("read bytes: %s, %s", tsUrl, err.Error())
	}
	sf := d.result.M3u8.Segments[segIndex]
	if sf == nil {
		return fmt.Errorf("invalid segment index: %d", segIndex)
	}
	/*获得此seg对应的key*/
	key, ok := d.result.Keys[sf.KeyIndex]
	if ok && key != "" {
		/*针对内容进行解密*/
		bytes, err = tool.AES128Decrypt(bytes, []byte(key),
			[]byte(d.result.M3u8.Keys[sf.KeyIndex].IV))
		if err != nil {
			return fmt.Errorf("decryt: %s, %s", tsUrl, err.Error())
		}
	}
	// https://en.wikipedia.org/wiki/MPEG_transport_stream
	// Some TS files do not start with SyncByte 0x47, they can not be played after merging,
	// Need to remove the bytes before the SyncByte 0x47(71).
	syncByte := uint8(71) //0x47
	bLen := len(bytes)
	for j := 0; j < bLen; j++ {
		if bytes[j] == syncByte {
			bytes = bytes[j:]
			break
		}
	}
	w := bufio.NewWriter(f)
	if _, err := w.Write(bytes); err != nil {
		return fmt.Errorf("write to %s: %s", fTemp, err.Error())
	}
	// Release file resource to rename file
	_ = f.Close()
	if err = os.Rename(fTemp, fPath); err != nil {
		return err
	}

	return nil
}

func (d *Downloader) next() (slice *FileSlice, end bool, err error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if len(d.queue) == 0 {
		err = fmt.Errorf("queue empty")
		if d.finish == int32(d.segLen) {
			/*队列为空，且均完成*/
			end = true
			return
		}
		// Some segment indexes are still running.
		end = false
		return
	}

	/*取队首的任务*/
	slice = d.queue[0]
	d.queue = d.queue[1:]
	return
}

/*将segIndex号job回弹，稍后重试*/
func (d *Downloader) back(segIndex int,tries int) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if sf := d.result.M3u8.Segments[segIndex]; sf == nil {
		return fmt.Errorf("invalid segment index: %d", segIndex)
	}
	d.queue = append(d.queue, &FileSlice{segId:segIndex,tries:tries,})
	return nil
}

/*执行文件合并*/
func (d *Downloader) merge() error {
	// In fact, the number of downloaded segments should be equal to number of m3u8 segments
	missingCount := 0
	for idx := 0; idx < d.segLen; idx++ {
		tsFilename := tsFilename(idx)
		f := filepath.Join(d.tsFolder, tsFilename)
		if _, err := os.Stat(f); err != nil {
			missingCount++
		}
	}
	if missingCount > 0 {
		fmt.Printf("[warning] %d files missing\n", missingCount)
	}

	// Create a TS file for merging, all segment files will be written to this file.
	//mFilePath := filepath.Join(d.folder, mergeTSFilename)
	mFilePath := filepath.Join(d.folder, d.fileName)
	mFile, err := os.Create(mFilePath)
	if err != nil {
		return fmt.Errorf("create main TS file failed：%s", err.Error())
	}
	//noinspection GoUnhandledErrorResult
	defer mFile.Close()

	writer := bufio.NewWriter(mFile)
	mergedCount := 0
	for segIndex := 0; segIndex < d.segLen; segIndex++ {
		tsFilename := tsFilename(segIndex)
		bytes, err := ioutil.ReadFile(filepath.Join(d.tsFolder, tsFilename))
		_, err = writer.Write(bytes)
		if err != nil {
			continue
		}
		mergedCount++
		tool.DrawProgressBar("merge",
			float32(mergedCount)/float32(d.segLen), progressWidth)
	}

	_ = writer.Flush()
	if mergedCount != d.segLen {
		fmt.Printf("[warning] \n%d files merge failed", d.segLen-mergedCount)
	} else {
		// Remove `ts` folder
		_ = os.RemoveAll(d.tsFolder)
		fmt.Printf("\n[output] %s\n", mFilePath)
    }

	return nil
}

func (d *Downloader) tsURL(segIndex int) string {
	seg := d.result.M3u8.Segments[segIndex]
	return tool.ResolveURL(d.result.URL, seg.URI)
}

func (d *Downloader) IsExist() bool {
	mFilePath := filepath.Join(d.folder, d.fileName)
	exist, err := path_exists(mFilePath)
	if err != nil {
		return false
	}

	return exist
}
func (d *Downloader) GetFileName() string {
	return d.fileName
}

func tsFilename(ts int) string {
	return strconv.Itoa(ts) + tsExt
}

func genSlice(len int) []*FileSlice {
	s := make([]*FileSlice, 0)
	for i := 0; i < len; i++ {
		s = append(s, &FileSlice{segId:i,tries:0,})
	}
	return s
}

func GenFileName(url string) string {
	url = strings.Replace(url, ":", "_", -1)
	url = strings.Replace(url, "/", "_", -1)
	url = strings.Replace(url, " ", "-", -1)
	return url + ".ts"
}
