package dl

import (
	"encoding/json"
	"os"
	"sync"
	//"fmt"
	"strings"
)

type State struct {
	Finish   bool   `json:"state"`
	SegIndex int    `json:"index"`
	TsUrl    string `json:"url"`
}

type FinishState struct {
	lock sync.Mutex
	//state map[int]bool
	//exState map[int]State
	state map[int]State
}

func path_exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func load(path string) (*FinishState, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	state := make(map[int]State)
	//state := make(map[int]bool)
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&state)
	if err != nil {
		return nil, err
	}

	return &FinishState{
		state: state,
	}, nil
}

func LoadFinishState(url string, path string) (*FinishState, error) {
	isExist, _ := path_exists(path)
	if isExist {
		return load(path)
	}

	f := &FinishState{
		state: make(map[int]State),
	}

	f.lock.Lock()
	f.state[-1] = State{
		Finish:   false,
		SegIndex: -1,
		TsUrl:    url,
	}
	defer f.lock.Unlock()
	if err := f.save(path); err != nil {
		return nil, err
	}

	return f, nil
}

func (f *FinishState) save(path string) error {
	//fmt.Println(f.state)
	fTemp := path + tsTempFileSuffix
	file, err := os.Create(fTemp)
	if err != nil {
		return err
	}
	//file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)

	//defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(f.state)
	if err != nil {
		return err
	}

	_ = file.Close()
	if err = os.Rename(fTemp, path); err != nil {
		return err
	}

	return nil
}

func (f *FinishState) isFinished(segIndex int) bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	meta, ok := f.state[segIndex]
	if !ok {
		return false
	}

	return meta.Finish
}

func (f *FinishState) isMatched(segIndex int, text string) (bool, string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	meta, ok := f.state[segIndex]
	if !ok {
		return false, ""
	}

	if strings.Contains(meta.TsUrl, text) {
		return true, meta.TsUrl
	}
	return false, ""
}

func (f *FinishState) updateFinishState(segIndex int, path string, tsUrl string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.state[segIndex] = State{
		Finish:   true,
		SegIndex: segIndex,
		TsUrl:    tsUrl,
	}
	return f.save(path)
	//return nil
}
