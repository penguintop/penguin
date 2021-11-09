import math
import requests
import json
import time
import os
import logging.handlers

########################################################################################################################
# create log directory
log_dir = os.path.dirname(os.path.abspath(__file__)) + os.sep + 'logs'
if not os.path.isdir(log_dir):
  os.makedirs(log_dir)

# set the TimeRoatingFileHandler
timefilehandler=logging.handlers.TimedRotatingFileHandler("logs/order.log", when='D', interval=1, backupCount=7)
timefilehandler.suffix="%Y-%m-%d.log"

# set the formattter
formatter = logging.Formatter('[%(asctime)s] [%(filename)s] [%(lineno)d] [%(levelname)s]: %(message)s')
timefilehandler.setFormatter(formatter)

# load logging and add timefilehandler to logger
logging.basicConfig()
logger = logging.getLogger(__name__)
logger.addHandler(timefilehandler)
logger.setLevel(logging.INFO)

########################################################################################################################
def load_config():
  global config
  with open("config.json", "r") as f:
    config = json.load(f)

def save_config():
  global  config
  with open("config.json", "w") as f:
    json.dump(config, f, indent=2)

########################################################################################################################
# json-rpc methods
def request(method,params):
  global config
  url = "http://%s:%s/api"%(config["rpc_addr"],config["rpc_port"])

  payload = "{\"jsonrpc\": \"2.0\", \"params\": %s,   \"id\": \"45\",   \"method\": \"%s\"    }"%(json.dumps(params),method)
  headers = {
    'content-type': "application/json",
    'authorization': "Basic YTpi"
  }

  response = requests.request("POST", url, data=payload, headers=headers)
  return response

def safe_request(method,params):
  while True:
    res = request(method,params)
    if res.status_code!=200 or "result" not in res.json():
      time.sleep(10)
      logger.debug('unkown rpc error code:%d,res:%s,req:'%(res.status_code,res.text),method,params)
    else:
      return res.json()

def handle_call_params(params):
  call_params = params
  if type(params) == type([]):
    sparams = [str(i) for i in params]
    call_params = ",".join(sparams)
  return call_params

def invoke_contract_offline(account_name, contract, method, params):
  return safe_request("invoke_contract_offline", [account_name, contract, method, handle_call_params(params)])

def query_contract_staking_count():
  res =invoke_contract_offline(config["caller_account_name"], config["staking_contract"], "info", "")
  return res["result"]

def invoke_contract(account_name, contract, method, params):
  return safe_request("invoke_contract", [account_name, "0.00001", 500000, contract, method, handle_call_params(params)])

def cal_need_amount(x):
  return (350*math.pow(1.008,-x/2000) + 50)*100000000

def check_transaction_on_chain(trxid):
  while True:
    try:
      res = safe_request("get_transaction", [trxid])
      if int(res["result"]["block_num"])>0:
        return True
      else:
        logger.warn("transaction unkown not onchain trxId:",trxid)
        time.sleep(10)
    except Exception as ex:
      logger.warn("transaction unkown not onchain exception:",ex)
      time.sleep(10)

if __name__ == "__main__":
  load_config()
  logger.info("##################################start priceOrder program##################################")
  logger.info("current time: %d" % int(time.time()))
  logger.info("latest time: %d" %  int(config["last_staking_price_update_time"]))

  diffSecs = 13*24*3600
  while True:
    if int(time.time()) - int(config["last_staking_price_update_time"]) <= diffSecs:
      time.sleep(30)
      logger.info("current time: %d" % int(time.time()))
      logger.info("latest time: %d" %  int(config["last_staking_price_update_time"]))
      continue

    # query the contract information
    res = query_contract_staking_count()
    obj = json.loads(res)
    if "totalMinerCount" not in obj:
      logger.warn("Can't found any keyword totalMinerCount")
      continue

    # price order now...
    amount = cal_need_amount(int(obj["totalMinerCount"]))
    print("setStakingNeedAmount",str(amount))
    res = invoke_contract(config["admin_account_name"], config["staking_contract"], "setStakingNeedAmount", str(amount))

    # check the transaction now...
    check_transaction_on_chain(res["result"]["trxid"])
    logger.info("update price success!")
    
    config["last_staking_price_update_time"] = int(time.time())
    save_config()
