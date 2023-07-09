package dl

import (
	"encoding/json"
	"os"
	"sync"
	//"fmt"
)

type FinishState struct {
	lock     sync.Mutex
	state map[int]bool
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

	state := make(map[int]bool)
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&state)
	if err != nil {
		return nil, err
	}

	return &FinishState{
		state: state,
	}, nil
}

func LoadFinishState(path string) (*FinishState, error) {
	isExist, _ := path_exists(path)
	if isExist {
		return load(path)
	}

	f := &FinishState{
		state: make(map[int]bool),
	}

	f.lock.Lock()
	f.state[-1] = false
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
	value, ok := f.state[segIndex]
	if !ok {
		return false
	}

	return value
}

func (f *FinishState) updateFinishState(segIndex int, path string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.state[segIndex] = true
	return f.save(path)
	//return nil
}
