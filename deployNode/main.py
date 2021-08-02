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

def invoke_contract(account_name,contract,method,params):
    return safe_request("invoke_contract",[account_name,"0.00001",500000,contract,method,handle_call_params(params)])

def deploy_contract(account_name,contract_path):
    res = safe_request("register_contract", [account_name, "0.00001", 500000, contract_path])
    return res["result"]["contract_id"]

def generate_block(i=1):
    time.sleep(6*i)

def query_height():
    res = safe_request("network_get_info",[])

    if res["result"]["target_block_height"] - res["result"]["current_block_height"]>1000:
        return 0
    return res["result"]["current_block_height"]

''' "fee": {
              "amount": 5100000,
              "asset_id": "1.3.0"
            },
            "invoke_cost": 500000,
            "gas_price": 1000,
            "caller_addr": "XWCNdbgFmQia2i58PcH918kSPMLrtwZ4kwK2V",
            "caller_pubkey": "02bc18900d005a1e832c4f4e0d41d90037e281d759441d9c8075d3c2c07b13d0b0",
            "contract_id": "XWCCbayhzZMXu1Q9ab2qzWSbG3MC9T1tR1fKB",
            "contract_api": "transfer",
            "contract_arg": "XWCNcZZKCMnRnr5JLp1gjxKV1dyRiuBZnGqjL,100000000000"
          }
'''

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

def create_swap(owner,amount):
    deploySimpleSwap_res = invoke_contract_offline(config["caller_account_name"],config["factory_addr"],"deploySimpleSwap",[owner])
    print("deploySimpleSwap",deploySimpleSwap_res)
    if deploySimpleSwap_res["result"] == "":
        # create swap transfer to owner ,transfer amount
        swap_addr = deploy_contract(config["admin_account_name"],"XRC20SimpleSwap.glua.gpc")

        res = invoke_contract(config["admin_account_name"],swap_addr,"init_config","%s,%s,%d"%(owner,config["token_addr"],0))
        print("init_config", res)
        init_trx_id = res["result"]["trxid"]

        generate_block(2)
        check_transaction_on_chain(init_trx_id)
        res = invoke_contract(config["receive_account_name"],config["token_addr"],"transfer","%s,%s"%(swap_addr,str(amount)))
        print("transfer",res)
        generate_block()
        transfer_trx_id = res["result"]["trxid"]
        check_transaction_on_chain(transfer_trx_id)

        res = invoke_contract(config["admin_account_name"],config["factory_addr"],"setSimpleSwap","%s,%s"%(owner,swap_addr))
        print("setSimpleSwap",res)
        simpleSwap_trx_id = res["result"]["trxid"]
        check_transaction_on_chain(simpleSwap_trx_id)

    else:
        # already register only transfer amount
        print("already create swap. only receive this amount,from:%s,amount:%s"%(owner,str(amount)))
        # res = invoke_contract(config["receive_account_name"],config["token_addr"],"transfer","%s,%s"%(deploySimpleSwap_res["result"],str(amount)))
        # print("setSimpleSwap", res)
        # transfer_trx_id = res["result"]["trxid"]
        # check_transaction_on_chain(transfer_trx_id)

def collect_block(height):
    res = safe_request("get_block",[height])
    index = 0
    for one_trx in res["result"]["transactions"]:
        for one_op in one_trx["operations"]:
            if one_op[0] == 79:
                if one_op[1]["contract_id"] == config["token_addr"] and one_op[1]["contract_api"]=="transfer":
                    args = one_op[1]["contract_arg"].split(',')
                    if len(args)==2 and args[0] == config["receive_addr"]:
                        transactionId = res["result"]["transaction_ids"][index]
                        events = safe_request("get_contract_invoke_object",[transactionId])["result"]
                        event = events[0]
                        if event["exec_succeed"] == True:
                            for one_event in event["events"]:
                                if one_event["event_name"] == "Transfer":
                                    transfer_obj = json.loads(one_event["event_arg"])
                                    if transfer_obj["to"] == config["receive_addr"]:
                                        amount = transfer_obj["amount"]
                                        from_addr = transfer_obj["from"]
                                        create_swap(from_addr,amount)
                                        # do create swap loop

        index +=1
    pass

def main_loop():
    global  config
    load_config()
    cur_height = query_height()
    while config["block_height"] < cur_height:
        config["block_height"] += 1
        collect_block(config["block_height"])
        if config["block_height"] % 1000 == 0:
            save_config()
            print("collect block target:%d,current:%d"%(cur_height,config["block_height"]))

    while True:
        cur_height = query_height()
        if config["block_height"] < cur_height:
            config["block_height"] += 1
            collect_block(config["block_height"])
        else:
            time.sleep(5)

if __name__ == "__main__":
    config=None
    #global config
    main_loop()



