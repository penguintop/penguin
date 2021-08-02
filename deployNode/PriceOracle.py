import math
import requests
import json
import time

def load_config():
    global config
    with open("config.json","r") as f:
        config = json.load(f)

def save_config():
    global  config
    with open("config.json","w") as f:
        json.dump(config,f)

def request(method,params):
    global config
    url = "http://%s:%s/api"%(config["rpc_addr"],config["rpc_port"])

    payload = "{\"jsonrpc\": \"2.0\", \"params\": %s,   \"id\": \"45\",   \"method\": \"%s\"    }"%(json.dumps(params),method)
    headers = {
        'content-type': "application/json",
        'authorization': "Basic YTpi"
        }

    response = requests.request("POST", url, data=payload, headers=headers)
    #print(payload)
    return response

def safe_request(method,params):
    while True:
        res = request(method,params)
        #print(res.text)
        if res.status_code!=200 or "result" not in res.json():
            time.sleep(5)
            print('unkown rpc error code:%d,res:%s,req:'%(res.status_code,res.text),method,params)
        else:
            return res.json()

def handle_call_params(params):
    call_params = params
    if type(params) == type([]):
        sparams = [str(i) for i in params]
        call_params = ",".join(sparams)
    return call_params

def invoke_contract_offline(account_name,contract,method,params):
    return safe_request("invoke_contract_offline",[account_name,contract,method,handle_call_params(params)])

def query_contract_staking_count():
    #"staking_contract": "", "last_staking_price_update_time":
    res =invoke_contract_offline(config["caller_account_name"],config["staking_contract"],"info","")
    return res["result"]

def invoke_contract(account_name,contract,method,params):
    return safe_request("invoke_contract",[account_name,"0.00001",500000,contract,method,handle_call_params(params)])

def cal_need_amount(x):
    return (350*math.pow(1.008,-x/2000) +50)*100000000

def check_transaction_on_chain(trxid):
    while True:
        try:
            res = safe_request("get_transaction", [trxid])
            if int(res["result"]["block_num"])>0:
                return True
            else:
                print("transaction unkown not onchain trxId:",trxid)
                time.sleep(5)
        except Exception as ex:
            print("transaction unkown not onchain exception:",ex)
            time.sleep(5)

if __name__ == "__main__":
    load_config()
    if  int(time.time())- int(config["last_staking_price_update_time"]/3600/24)*24*3600>13*24*3600:
        res = query_contract_staking_count()
        obj = json.loads(res)
        if "totalMinerCount" in obj:
            amount = cal_need_amount(int(obj["totalMinerCount"]))
            print("setStakingNeedAmount",str(amount))
            res = invoke_contract(config["admin_account_name"],config["staking_contract"],"setStakingNeedAmount",str(amount))

            check_transaction_on_chain(res["result"]["trxid"])
            print("update price success!")
            config["last_staking_price_update_time"] = int(time.time())
            save_config()


