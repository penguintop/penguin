# XWC Penguin Pen

## Contributing

Please read the [coding guidelines](CODING.md).

## License

This library is distributed under the BSD-style license found in the [LICENSE](LICENSE) file.

# Installing

## OS requirements

Recommends to use Unix based system for installation

- Ubuntu 20.04
- MacOS

## Software

In this installation manual we will use following additional software. You need to install it before

- git for cloning repository, for install try `sudo apt install git`
- screen agent for using sessions, for install try `sudo apt install screen`

# Preparation

Because this manual is for **completely full newbies** *(but of course they need some knowledge about Linux and terminal)* we'll tell about very obvious things

In this manual we will use the following folder structure, so you should to prepare all folders and add the necessary permissions for them.

- Create folder`mkdir /srv/pen/`
- Add permissions`chmod +x /srv/pen/` && `cd pen`
- Download git repository `git clone [https://github.com/penguintop/penguin](https://github.com/penguintop/penguin)` and it's create the folder `penguin` automatically via downloading
- Add permissions`chmod +x /srv/pen/penguin/`
- Create folder`mkdir /srv/pen/penguin/pen_data/`
- Add permissions`chmod +x /srv/pen/penguin/pen_data/`
- Go to the folder `cd /srv/pen/penguin/`
- download PEN binary prepared file  `wget "https://www.penguin.top/download/pen"`
- Add permissions`chmod +x /srv/pen/penguin/pen`

# Run Node

Check `pwd`

it should shows that you stays in ``/srv/pen/penguin/`` Penguin folder witn `pen` binary file

This command run node with using screen session

`screen -S pen-test ./pen start --data-dir=./data --audit-mode --audit-endpoint=http://123.129.224.30:29999 --swap-endpoint=http://123.129.224.30:29900 --debug-api-enable --full-node --cache-capacity=10000000 --bootnode=/dnsaddr/penguin.top --cors-allowed-origins=*`

# How to upload file.pdf,jpg,png,etc to PEN using GUI

Here is WEB GUI for Penguin [http://112.47.58.10:8888/](http://112.47.58.10:8888/)

It contains following sections

- Settings
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLXpsYtBRyjGyyvJiHlTsQm-mTgAVimsoC5VX61zZxY3-fo0ESlWlCdbzp8dXugUi07jUiIOdar8D7HpQOv-Mb6rRdcfN9aV6_nozXA3yaFL35ko_qkAUrCbx13fCfYDQQkhx2bd9XKVoSRUJO83_lv-=w752-h498-no?authuser=0?raw=true)

You can use DEMO IP of PEN Node[`http://112.47.58.10:1633`](http://112.47.58.10:1633/) for API and [`http://112.47.58.10:1635`](http://112.47.58.10:1635/) for Debug

Or you can use own server or pc `IP` of your PEN node.

- Status

Here is information about your connected node
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLVcN_0ogZNJxfYzHjFoMbdR-UMUofveCNdlSMWGF3NsVWmcuFjfjcjUg3B7GdeQqit3q8NwkAOOFr9St3kGlWpTJMMzP4NuM86gz5lLL-Ovw_tabm9x7TbmaKBjJ6k_aQlVrNEJc78ainympAGEOm_o=w994-h1213-no?authuser=0?raw=true)

- Files

Here is upload/download manager
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLVcmcLRB3jjYcWNfGdDQ3AOLjfGzgod_iyi9_m_m_I__m7UnCnOOEDsMS7x9_kjuNzPeJ-K7gLVlrrV5CKd_zBw4WRkPW1zVP453zNNHFLdp6g6jZHpDf2uEw5-78UYOR_PTVD_cfqA4PIw-tWYFJ2E=w994-h622-no?authuser=0?raw=true)

When you add the file you will see in left bottom corner notification about it, check the screenshot
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLW-1k_8zU_EEJj426efJv3IUi-ihWK1eIZHk813cCZc62i2a07EXYAjzDuXNWvmjVUjQhCxZ1WkiqPjKCDU1DRHPxjYVILXDNPQS6j_7zodvKX_nK0lMbkfP02zMv3boMj_PGKbZ0ueBUGgEkN-r6r2=w994-h1398-no?authuser=0?raw=true)

after it you should click button "Upload" below.

Than you'll see something like on this gif video, hash of your file.
https://i.gyazo.com/80eb8aeaacc3e4b784e5a4ccdeb13321.mp4


When you go to the "Download" tab and the copy paste hash of your file and click search. You'll get the link to your file in the Penguin PEN blockchain. Please see illustration in the video below
https://i.gyazo.com/390210ddff095523d391b59d24c4868c.mp4


- Stamps
- Accounting
- Peers

# Backup

- [ ]  backup private key

After entering and confirming the password, the terminal will print out the private key of the XWC account, which needs to be backed up
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLUWiBZJW0rCHXpYlzxY0UpLqFG2G7dAV7yN5QGT0cSgr9H-yk8sDjYAHL4ix3tnCDAbPD5L29Glz_a0KAHK6aDkDF9h5ilLlNSsTaQglRwS9l5jM5Nz7V7OV20qZz0Jfsg5GxotA1ZuEz_XxK8LT9Yn=w993-h218-no?authuser=0?raw=true)


# Fill up an node wallet

3. After the private key backup is completed, press any key, you can see the following print
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLWLG6J3qH-8uRnhS-fqiWngU-cFNaomp-z6oCC1OuAhgIuZo-o8RDQd0jg2COlYjKtwg1CAIYS-Y8yqQJZjIGCAzu4CurpW-ZCn47iY8cBFsjJHGyJ5lxOh7buycdKS_fcmAZTLwHVu0Lpd1ZPVdtSd=w2107-h433-no?authuser=0?raw=true)


- [ ]  If you do not participate in PEN mining, you need to transfer 1 XWC (for handling fees) and no less than 1 PEN to the relevant address
That's it.
- [ ]  If you need to participate in PEN mining, you need to transfer 1 XWC (for handling fee) and no less than 401 XWC to the relevant address
PEN.

# Check status of installation

4. If you can see the following prints later, it means that the PEN node has been started normally
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLVIAE05XpRJ_OHDeG0wfSaeYcBwVKVjyHeIvYwXJQ2zgimXJo2uTc1Ux0in9yyZ82X3XQOHGOTWKf-Q8GgzQV-SguWXDt57xyOO-5IkNFoV1jePYI-XVqGZkxKDNyeI6DBUEIq3rvILPm-h4C9qL4Ur=w993-h485-no?authuser=0?raw=true)


5. If you see the following print, it means that the PEN node has turned on the mining mode
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLUxl6iPD1EGi0gTM4VGlnvibwBl5Vqv00BYn-yxTkH010T-hkIV2R7AbxJiaY2Q4LAdzg07_zjXXd0ilcIV6YeOlWyZ5I9tNMZNWsVzj7siML7owpdH5dvxWNW4LktnWB2lNGZX2_BzaOvkfA9Y0XpJ=w993-h252-no?authuser=0?raw=true)

# Backups additional

6. How to export and backup PEN's XWC account private key if you forget to back up the private key when running PEN for the first time

`./pen dumpkey --data-dir=/home/pen/pen_data`
![alt text](https://lh3.googleusercontent.com/pw/AM-JKLVh_47pEIn9t0OKbUgXVDMgFlhvG3xLbd2v5Zev8iUf1OPThNVX561tcUF-h-NUWlKgmj_kjzxDGFHALKuVFgoKPdrr-EByrEgv3LMKM24UVmCXOLqmxSkFGOtP19T2Ek0ImCUxXE4RoZcuLFS15E9F=w993-h140-no?authuser=0?raw=true)


# PEN startup parameters:

1. `--audit-mode`
Open mining mode
2. `--audit-endpoint`
Mining authentication server endpoint
3. `--data-dir`
Local data directory
4. `--swap-endpoint`
XWC wallet node endpoint
5. `--debug-api-enable`
Open debug api

# Receive income

Website: [https://www.penguin.top/home](https://www.penguin.top/home)
Receiving operation:

1. Enter the wallet address, click the [Query] button, and the amount will pop up;
