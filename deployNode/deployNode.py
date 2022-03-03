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
timefilehandler=logging.handlers.TimedRotatingFileHandler("logs/deploy.log", when='D', interval=1, backupCount=7)
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
# configuration file
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
def request(method, params):
  global config
  url = "http://%s:%s/api"%(config["rpc_addr"], config["rpc_port"])

  payload = "{\"jsonrpc\": \"2.0\", \"params\": %s,   \"id\": \"45\",   \"method\": \"%s\"    }"%(json.dumps(params),method)
  headers = {
    'content-type': "application/json",
    'authorization': "Basic YTpi"
  }

  response = requests.request("POST", url, data=payload, headers=headers)
  return response

def safe_request(method, params):
  while True:
    try:
      res = request(method, params)
      if res.status_code != 200 or "result" not in res.json():
        time.sleep(10)
        logger.debug('unkown rpc error code:%d, res:%s, req:' % (res.status_code, res.text), method, params)
      else:
        return res.json()

    except Exception as ex:
      logger.warn(ex)
      time.sleep(10)

def handle_call_params(params):
  call_params = params
  if type(params) == type([]):
    sparams = [str(i) for i in params]
    call_params = ",".join(sparams)
  return call_params

def invoke_contract_offline(account_name, contract, method, params):
  return safe_request("invoke_contract_offline", [account_name, contract, method, handle_call_params(params)])

def invoke_contract(account_name,contract,method,params):
  return safe_request("invoke_contract", [account_name, "0.00001", 500000, contract, method, handle_call_params(params)])

def deploy_contract(account_name,contract_path):
  res = safe_request("register_contract", [account_name, "0.00001", 500000, contract_path])
  return res["result"]["contract_id"]

def generate_block(i = 1):
  time.sleep(6*i)

def query_height():
  res = safe_request("network_get_info",[])
  if (res["result"]["target_block_height"] - res["result"]["current_block_height"]) > 1000:
    return 0

  return res["result"]["current_block_height"]

def check_transaction_on_chain(trxid):
  while True:
    try:
      res = safe_request("get_transaction", [trxid])
      if int(res["result"]["block_num"]) > 0:
        return True
      else:
        logger.warn("transaction unkown not onchain trxId:", trxid)
        time.sleep(10)
    except Exception as ex:
      logger.warn("transaction unkown not onchain exception:", ex)
      time.sleep(10)

def create_swap(owner, amount):
  # che user's contract is registered or not
  deploySimpleSwap_res = invoke_contract_offline(config["caller_account_name"], config["factory_addr"], "deploySimpleSwap", [owner])
  if deploySimpleSwap_res["result"] != "":
    logger.info("already create swap. only receive this amount,from:%s,amount:%s"%(owner,str(amount)))
    return

  # create swap contract
  swap_addr = deploy_contract(config["admin_account_name"], "XRC20SimpleSwap.glua.gpc")
  if swap_addr == "":
    logger.warn("can't create contract for user: %s %s" % (owner, str(amount)))
    return

  # change the owner of contract
  generate_block(2)
  res = invoke_contract(config["admin_account_name"], swap_addr, "init_config", "%s,%s,%d" % (owner, config["token_addr"], 0))
  init_trx_id = res["result"]["trxid"]
  if init_trx_id == "":
    logger.warn("init_config is failed for user: %s %s %s" % (owner, str(amount), swap_addr))
    return

  # transfer to contract
  generate_block(2)
  check_transaction_on_chain(init_trx_id)
  res = invoke_contract(config["receive_account_name"], config["token_addr"], "transfer", "%s,%s" % (swap_addr, str(amount)))

  generate_block(2)
  transfer_trx_id = res["result"]["trxid"]
  check_transaction_on_chain(transfer_trx_id)

  res = invoke_contract(config["admin_account_name"], config["factory_addr"], "setSimpleSwap", "%s,%s" % (owner, swap_addr))
  logger.info("setSimpleSwap",res)

  simpleSwap_trx_id = res["result"]["trxid"]
  check_transaction_on_chain(simpleSwap_trx_id)

def collect_block(height):
  index = -1
  res = safe_request("get_block", [height])
  for one_trx in res["result"]["transactions"]:
    index += 1
    for one_op in one_trx["operations"]:
      if one_op[0] != 79:
        continue

      if one_op[1]["contract_id"] != config["token_addr"] or one_op[1]["contract_api"] != "transfer":
        continue

      args = one_op[1]["contract_arg"].split(',')
      if len(args) != 2 or args[0] != config["receive_addr"]:
        continue

      transactionId = res["result"]["transaction_ids"][index]
      events = safe_request("get_contract_invoke_object",[transactionId])["result"]
      event = events[0]
      if event["exec_succeed"] == False:
        continue

      for one_event in event["events"]:
        if one_event["event_name"] != "Transfer":
          continue

        transfer_obj = json.loads(one_event["event_arg"])
        if transfer_obj["to"] == config["receive_addr"]:
          amount = transfer_obj["amount"]
          from_addr = transfer_obj["from"]
          create_swap(from_addr, amount)

def main_loop():
  global config
  load_config()

  indexs = 0
  cur_height = query_height()
  logger.info("collect block target:%d, current:%d" % (cur_height, config["block_height"]))
  while True:
    # sleep and wait new block
    if config["block_height"] >= cur_height:
      save_config()
      time.sleep(10)
      cur_height = query_height()
      continue

    # handle the block
    indexs += 1
    config["block_height"] += 1
    collect_block(config["block_height"])
    if indexs >= 10:
      indexs = 0
      save_config()
      logger.info("collect block target:%d, current:%d" % (cur_height, config["block_height"]))

if __name__ == "__main__":
  config=None
  logger.info("##############################start deploy program##############################")

  main_loop()
