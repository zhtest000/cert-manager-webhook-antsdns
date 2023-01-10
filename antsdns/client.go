package antsdns

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s.io/klog"
	"net/http"
	//"github.com/jetstack/cert-manager/pkg/issuer/acme/dns/util"
	"net/http/httputil"
	"sort"
	"strconv"
	"time"
)

type Client struct {
	dnsc *antsClient
}

func newClient(ispAddressV string,apiIdV string,apiKeyV string) *antsClient {
	return &antsClient{
		ispAddress: ispAddressV,
		appId: apiIdV,
		appKey: apiKeyV,
		dumpRequestResponse:false,
	}
}

func (c *antsClient) antsRecordsUrl(method string) string {
	//return fmt.Sprintf("%s/domains/%s/records", antsLiveDnsBaseUrl, domain)
	return fmt.Sprintf( c.ispAddress+"/ants-dns-api/app/capi/"+method)
}



func (c *antsClient) doRequest(req *http.Request, readResponseBody bool) (int, []byte, error) {
	if c.dumpRequestResponse {
		dump, _ := httputil.DumpRequest(req, true)
		fmt.Printf("Request: %q\n", dump)
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	if c.dumpRequestResponse {
		dump, _ := httputil.DumpResponse(res, true)
		fmt.Printf("Response: %q\n", dump)
	}

	if res.StatusCode == http.StatusOK && readResponseBody {
		data, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return 0, nil, err
		}
		return res.StatusCode, data, nil
	}

	return res.StatusCode, nil, nil
}

func getDictSign(dictMap map[string]string)string{
	signStr := ""
	keys := make([]string, 0)
	for k, _ := range dictMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		//fmt.Println(k, dictMap[k])
		if k!="sign"{
			if len(dictMap[k])>0 {
				if 0==len(signStr) {
					signStr += k + "=" + dictMap[k]
				}else {
					signStr +="&"+ k + "=" + dictMap[k]
				}
			}
		}
	}
	//fmt.Print(signStr)
	md5 := md5.New()
	md5.Write([]byte(signStr))
	sign := hex.EncodeToString(md5.Sum(nil))
	return sign
}

func (c *antsClient) HasTxtRecord(domain *string, name *string) (bool, error) {
	recordId,err := c.GetTxtRecordId(domain,name)
	if err != nil {
		return false, err
	}
	if len(recordId)>0 {
		return true,nil
	}else {
		klog.Errorf("not found recordId: %v", err)
		return false,nil
	}
}

func (c *antsClient) CreateTxtRecord(domain *string, top *string, value *string, ttl int) error {
	var dictMap map[string]string
	dictMap=make(map[string]string)

	dictMap["secretId"]=c.appId
	dictMap["secretKey"]=c.appKey
	dictMap["domain"]= *domain
	dictMap["top"]=*top
	dictMap["line"]="默认"
	dictMap["value"]=*value
	dictMap["mx"]="1"
	dictMap["record_type"]="TXT"
	dictMap["ttl"]=  strconv.Itoa(ttl)
	dictMap["weight"]="1"
	sign:=getDictSign(dictMap)
	dictMap["sign"]=sign
	delete(dictMap,"secretKey")
	body, err := json.Marshal(dictMap)
	klog.Infof("CreateTxtRecord Presented txt record %v",string(body))
	if err != nil {
		return fmt.Errorf("cannot marshall to json: %v", err)
	}

	url:=c.antsRecordsUrl("Record.Create")
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	status, _, err := c.doRequest(req, false)
	if err != nil {
		return err
	}

	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("failed creating TXT record: %v", err)
	}

	return nil
}

func (c *antsClient) UpdateTxtRecord(domain *string, name *string, value *string, ttl int) error {
	recordId,err := c.GetTxtRecordId(domain,name)
	if nil!=err{
		return err
	}
	var dictMap map[string]string
	dictMap=make(map[string]string)
	dictMap["secretId"]=c.appId
	dictMap["secretKey"]=c.appKey
	dictMap["domain"]= *domain
	dictMap["top"]=*name
	dictMap["line"]="默认"
	dictMap["value"]=*value
	dictMap["mx"]="1"
	dictMap["record_type"]="TXT"
	dictMap["ttl"]=  strconv.Itoa(ttl)
	dictMap["weight"]="1"
	dictMap["record_id"] = recordId
	sign:=getDictSign(dictMap)
	dictMap["sign"]=sign
	delete(dictMap,"secretKey")
	body, err := json.Marshal(dictMap)
	if err != nil {
		return fmt.Errorf("cannot marshall to json: %v", err)
	}

	url:=c.antsRecordsUrl("Record.Modify")
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	status, _, err := c.doRequest(req, false)
	if err != nil {
		return err
	}

	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("failed creating TXT record: %v", err)
	}
	return nil

}

func (c *antsClient) GetTxtRecordId(domain *string, name *string) (string,error)  {
	var dictMap map[string]string
	dictMap=make(map[string]string)

	dictMap["secretId"]=c.appId
	dictMap["secretKey"]=c.appKey
	dictMap["domain"]= *domain
	dictMap["keyword"] = *name
	sign:=getDictSign(dictMap)
	dictMap["sign"]=sign
	delete(dictMap,"secretKey")
	body, err := json.Marshal(dictMap)
	if err != nil {
		klog.V(6).Infof("cannot marshall to json: %v", err)
		return "",fmt.Errorf("cannot marshall to json: %v", err)
	}
	url:=c.antsRecordsUrl("Record.List")
	//fmt.Printf("url:%s\r\n",url)
	//fmt.Printf("body:%s\r\n",string(body))
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	status, data, err := c.doRequest(req, true)
	if err != nil {
		return "", err
	}
	if status == http.StatusNotFound {
		return "", nil
	} else if status == http.StatusOK {
		// Maybe parse response body here to really ensure that the record is present
		//fmt.Println(string(data))
		var returnObj antsReturnObj
		err = json.Unmarshal(data, &returnObj)
		if 1==returnObj.Code{
			recordInfos:=returnObj.Data
			for _,recordStr:= range recordInfos{
				//fmt.Println(recordStr)
				var record antsRecordValue
				err = json.Unmarshal([]byte(recordStr), &record)
				//fmt.Println(record.Top,record.RecordType)
				//fmt.Println(err)
				if nil==err && "TXT"==record.RecordType{
					//fmt.Println(record.RecordId)
					return record.RecordId, nil
				}
			}
		}
		return "", nil
	} else {
		return "", fmt.Errorf("unexpected HTTP status: %d", status)
	}
}

func (c *antsClient) DeleteTxtRecord(domain *string, name *string) error {

	recordId,err := c.GetTxtRecordId(domain,name)
	if nil!=err{
		return err
	}
	//fmt.Printf("----recordId=%s\r\n",recordId)
	var dictMap map[string]string
	dictMap=make(map[string]string)

	dictMap["secretId"]=c.appId
	dictMap["secretKey"]=c.appKey
	dictMap["domain"]= *domain
	dictMap["record_id"]=recordId
	sign:=getDictSign(dictMap)
	dictMap["sign"]=sign
	delete(dictMap,"secretKey")
	body, err := json.Marshal(dictMap)
	if err != nil {
		return fmt.Errorf("cannot marshall to json: %v", err)
	}

	url:=c.antsRecordsUrl("Record.Remove")
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	status, data, err := c.doRequest(req, true)
	if err != nil {
		return err
	}

	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("failed creating TXT record: %v", err)
	}
	fmt.Printf(string(data))
	return nil
}

//func main() {
//	secretId := "abc1b943"
//	secretKey := "83801697b9684cdca7aec4b4c08ccd43"
//	antsClient := newClient(secretId,secretKey)
//	domain:="vedns.com"
//	name:="axtaaest"
//	value:="caaaaaaaaaavalue"
//	//antsClient.CreateTxtRecord(&domain,&name,&value,600)
//	//has,_:= antsClient.HasTxtRecord(&domain,&name)
//	antsClient.UpdateTxtRecord(&domain,&name,&value,600)
//	//antsClient.DeleteTxtRecord(&domain,&name)
//
//}
