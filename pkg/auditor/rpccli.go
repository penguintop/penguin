package auditor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func RequestServerTimestamp(baseUrl string, timeout uint64) (int64, error) {
	url := fmt.Sprintf("%s/api/getTime", baseUrl)

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    time.Duration(timeout) * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	type RespServerTimestampJson struct {
		Code int    `json:"code"`
		Data int64  `json:"data"`
		Msg  string `json:"msg"`
	}

	var respServerTimestamp RespServerTimestampJson
	err = json.Unmarshal(res, &respServerTimestamp)
	if err != nil {
		return 0, err
	}

	if respServerTimestamp.Code == 0 {
		return 0, errors.New(respServerTimestamp.Msg)
	}

	return respServerTimestamp.Data, nil
}

func RequestTask(baseUrl string, timeout uint64, timestamp int64, xwcAddr string, xwcPubKey string, penguinNode string, signature string) (uint64, error) {
	url := fmt.Sprintf("%s/api/getTask", baseUrl)

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    time.Duration(timeout) * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	type RequestTaskJson struct {
		Timestamp     int64  `json:"timestamp"`
		XwcAddr       string `json:"xwc_addr"`
		XwcSignPubkey string `json:"xwc_sign_pubkey"`
		PenguinAddr     string `json:"penguin_addr"`
		SignMsg       string `json:"sign_msg"`
	}

	requestTaskJson := RequestTaskJson{
		Timestamp:     timestamp,
		XwcAddr:       xwcAddr,
		XwcSignPubkey: xwcPubKey,
		PenguinAddr:     penguinNode,
		SignMsg:       signature,
	}
	requestTaskBuf, err := json.Marshal(requestTaskJson)
	if err != nil {
		return 0, err
	}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(requestTaskBuf)))
	if err != nil {
		return 0, err
	}

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	type TaskJson struct {
		TaskId uint64 `json:"TaskId"`
	}

	type RespTaskJson struct {
		Code int      `json:"code"`
		Data TaskJson `json:"data"`
		Msg  string   `json:"msg"`
	}

	var respTaskJson RespTaskJson
	err = json.Unmarshal(res, &respTaskJson)
	if err != nil {
		return 0, err
	}

	if respTaskJson.Code == 0 {
		return 0, errors.New(respTaskJson.Msg)
	}

	return respTaskJson.Data.TaskId, nil
}

func RequestReportMerkleRoot(baseUrl string, timeout uint64, taskId uint64, xwcAddr string, xwcPubKey string, penguinNode string, signature string,
	pathData [][]string) (uint64, uint64, error) {
	url := fmt.Sprintf("%s/api/reportMerkleRoot", baseUrl)

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    time.Duration(timeout) * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	type RequestReportMerkleRootJson struct {
		TaskId        uint64     `json:"task_id"`
		XwcAddr       string     `json:"xwc_addr"`
		XwcSignPubkey string     `json:"xwc_sign_pubkey"`
		PenguinAddr     string     `json:"penguin_addr"`
		SignMsg       string     `json:"sign_msg"`
		PathData      [][]string `json:"path_data"`
	}

	requestReportMerkleRootJson := RequestReportMerkleRootJson{
		TaskId:        taskId,
		XwcAddr:       xwcAddr,
		XwcSignPubkey: xwcPubKey,
		PenguinAddr:     penguinNode,
		SignMsg:       signature,
		PathData:      pathData,
	}
	requestReportMerkleRootBuf, err := json.Marshal(requestReportMerkleRootJson)
	if err != nil {
		return 0, 0, err
	}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(requestReportMerkleRootBuf)))
	if err != nil {
		return 0, 0, err
	}

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	type TaskPathJson struct {
		TaskId  uint64 `json:"TaskId"`
		PathInt uint64 `json:"PathInt"`
	}

	type RespReportMerkleRootJson struct {
		Code int          `json:"code"`
		Data TaskPathJson `json:"data"`
		Msg  string       `json:"msg"`
	}

	var respReportMerkleRootJson RespReportMerkleRootJson
	err = json.Unmarshal(res, &respReportMerkleRootJson)
	if err != nil {
		return 0, 0, err
	}

	if respReportMerkleRootJson.Code == 0 {
		return 0, 0, errors.New(respReportMerkleRootJson.Msg)
	}

	return respReportMerkleRootJson.Data.TaskId, respReportMerkleRootJson.Data.PathInt, nil
}

func RequestReportPathData(baseUrl string, timeout uint64, taskId uint64, xwcAddr string, xwcPubKey string, penguinNode string, signature string,
	pathData [][]string, penguinData string) error {
	url := fmt.Sprintf("%s/api/reportPathData", baseUrl)

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    time.Duration(timeout) * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	type RequestReportPathDataJson struct {
		TaskId        uint64     `json:"task_id"`
		XwcAddr       string     `json:"xwc_addr"`
		XwcSignPubkey string     `json:"xwc_sign_pubkey"`
		PenguinAddr     string     `json:"penguin_addr"`
		SignMsg       string     `json:"sign_msg"`
		PathData      [][]string `json:"path_data"`
		PenguinData     string     `json:"penguin_data"`
	}

	requestReportPathDataJson := RequestReportPathDataJson{
		TaskId:        taskId,
		XwcAddr:       xwcAddr,
		XwcSignPubkey: xwcPubKey,
		PenguinAddr:     penguinNode,
		SignMsg:       signature,
		PathData:      pathData,
		PenguinData:     penguinData,
	}

	requestReportPathDataBuf, err := json.Marshal(requestReportPathDataJson)
	if err != nil {
		return err
	}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(requestReportPathDataBuf)))
	if err != nil {
		return err
	}

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	type RespReportPathDataJson struct {
		Code int    `json:"code"`
		Data string `json:"data"`
		Msg  string `json:"msg"`
	}

	var respReportPathDataJson RespReportPathDataJson
	err = json.Unmarshal(res, &respReportPathDataJson)
	if err != nil {
		return err
	}

	if respReportPathDataJson.Code == 0 {
		return errors.New(respReportPathDataJson.Msg)
	}

	return nil
}
