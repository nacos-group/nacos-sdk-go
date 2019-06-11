package cache

import (
	"encoding/json"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func mkdirIfNecessary(path string) {
	if ok, _ := exists(path); !ok {
		err := os.Mkdir(path, 0755)
		if err != nil {
			log.Printf("[ERROR]:can not create dir:%s err:%v \n", path, err)
		}
	}
}

func GetFileName(cacheKey string, cacheDir string) string {
	return cacheDir + string(os.PathSeparator) + cacheKey
}

func WriteToFile(service model.Service, cacheDir string) {
	mkdirIfNecessary(cacheDir)
	sb, _ := json.Marshal(service)
	domFileName := GetFileName(utils.GetServiceCacheKey(service.Name, service.Clusters), cacheDir)

	err := ioutil.WriteFile(domFileName, sb, 0666)
	if err != nil {
		log.Printf("[ERROR]:faild to write cache:%s ,value:%s ,err:%s \n", domFileName, string(sb), err.Error())
	}

}

func ReadFromFile(cacheDir string) map[string]model.Service {
	files, err := ioutil.ReadDir(cacheDir)
	if err != nil {
		log.Printf("[ERROR]:read cacheDir:%s failed!err:%s \n", cacheDir, err.Error())
		return nil
	}
	serviceMap := map[string]model.Service{}
	for _, f := range files {
		fileName := GetFileName(f.Name(), cacheDir)
		b, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Printf("[ERROR]:failed to read cache file:%s,err:%s! ", fileName, err.Error())
			continue
		}

		s := string(b)
		service := utils.JsonToService(s)

		if service == nil {
			continue
		}

		serviceMap[f.Name()] = *service
	}

	log.Printf("finish loading cache, total: " + strconv.Itoa(len(files)))
	return serviceMap
}
