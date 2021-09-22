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

Because this manual for completely newbies we'll tell about about very obvious things
In this manual we will use following folder structure, so you should to prepare all folders and add the necessary permissions for them.
- Create folder`mkdir /srv/pen/`
- Add permissions`chmod +x /srv/pen/`
- Create folder`mkdir /srv/pen/penguin/`
- Add permissions`chmod +x /srv/pen/penguin/`
- Create folder`mkdir /srv/pen/penguin/pen_data/`
- Add permissions`chmod +x /srv/pen/penguin/pen_data/`
- Go to the folder `cd /srv/pen/penguin/`
- download PEN binary prepared file  `wget "https://www.penguin.top/download/pen"`
- Add permissions`chmod +x /srv/pen/penguin/pen`

# Run Node

2.0.1. Create folder for future PEN software `mkdir /srv/pen/`

2.0.2. Download git repository `git clone [https://github.com/penguintop/penguin](https://github.com/penguintop/penguin)`

2.1.0Go to the cloned Repo `cd /srv/pen/penguin/` 

- 2.1.1.then download PEN `wget "https://www.penguin.top/download/pen"`
- 2.1.2.add write permissions `chmod +x /srv/pen/penguin/pen`

2.2. please create folder PEN_DATA, for i.e. `mkdir /srv/pen/penguin/pen_data` and add write permissions `chmod +x /srv/pen/penguin/pen_data`

2.3.Start the PEN node

`./pen start --audit-mode --audit-endpoint=http://192.168.1.73:5555 --data-dir=/srv/pen/penguin/pen_data --swap-endpoint=http://192.168.1.123:19890/rpc --debug-api-enable`

# Backup

- [ ]  backup private key

After entering and confirming the password, the terminal will print out the private key of the XWC account, which needs to be backed up

[https://lh3.googleusercontent.com/pw/AM-JKLUWiBZJW0rCHXpYlzxY0UpLqFG2G7dAV7yN5QGT0cSgr9H-yk8sDjYAHL4ix3tnCDAbPD5L29Glz_a0KAHK6aDkDF9h5ilLlNSsTaQglRwS9l5jM5Nz7V7OV20qZz0Jfsg5GxotA1ZuEz_XxK8LT9Yn=w993-h218-no?authuser=0](https://lh3.googleusercontent.com/pw/AM-JKLUWiBZJW0rCHXpYlzxY0UpLqFG2G7dAV7yN5QGT0cSgr9H-yk8sDjYAHL4ix3tnCDAbPD5L29Glz_a0KAHK6aDkDF9h5ilLlNSsTaQglRwS9l5jM5Nz7V7OV20qZz0Jfsg5GxotA1ZuEz_XxK8LT9Yn=w993-h218-no?authuser=0)

# Fill up an node wallet

3. After the private key backup is completed, press any key, you can see the following print

[https://lh3.googleusercontent.com/pw/AM-JKLWLG6J3qH-8uRnhS-fqiWngU-cFNaomp-z6oCC1OuAhgIuZo-o8RDQd0jg2COlYjKtwg1CAIYS-Y8yqQJZjIGCAzu4CurpW-ZCn47iY8cBFsjJHGyJ5lxOh7buycdKS_fcmAZTLwHVu0Lpd1ZPVdtSd=w2107-h433-no?authuser=0](https://lh3.googleusercontent.com/pw/AM-JKLWLG6J3qH-8uRnhS-fqiWngU-cFNaomp-z6oCC1OuAhgIuZo-o8RDQd0jg2COlYjKtwg1CAIYS-Y8yqQJZjIGCAzu4CurpW-ZCn47iY8cBFsjJHGyJ5lxOh7buycdKS_fcmAZTLwHVu0Lpd1ZPVdtSd=w2107-h433-no?authuser=0)

- [ ]  If you do not participate in PEN mining, you need to transfer 1 XWC (for handling fees) and no less than 1 PEN to the relevant address
That's it.
- [ ]  If you need to participate in PEN mining, you need to transfer 1 XWC (for handling fee) and no less than 401 XWC to the relevant address
PEN.

# Check status of installation

4. If you can see the following prints later, it means that the PEN node has been started normally

[https://lh3.googleusercontent.com/pw/AM-JKLVIAE05XpRJ_OHDeG0wfSaeYcBwVKVjyHeIvYwXJQ2zgimXJo2uTc1Ux0in9yyZ82X3XQOHGOTWKf-Q8GgzQV-SguWXDt57xyOO-5IkNFoV1jePYI-XVqGZkxKDNyeI6DBUEIq3rvILPm-h4C9qL4Ur=w993-h485-no?authuser=0](https://lh3.googleusercontent.com/pw/AM-JKLVIAE05XpRJ_OHDeG0wfSaeYcBwVKVjyHeIvYwXJQ2zgimXJo2uTc1Ux0in9yyZ82X3XQOHGOTWKf-Q8GgzQV-SguWXDt57xyOO-5IkNFoV1jePYI-XVqGZkxKDNyeI6DBUEIq3rvILPm-h4C9qL4Ur=w993-h485-no?authuser=0)

5. If you see the following print, it means that the PEN node has turned on the mining mode

[https://lh3.googleusercontent.com/pw/AM-JKLUxl6iPD1EGi0gTM4VGlnvibwBl5Vqv00BYn-yxTkH010T-hkIV2R7AbxJiaY2Q4LAdzg07_zjXXd0ilcIV6YeOlWyZ5I9tNMZNWsVzj7siML7owpdH5dvxWNW4LktnWB2lNGZX2_BzaOvkfA9Y0XpJ=w993-h252-no?authuser=0](https://lh3.googleusercontent.com/pw/AM-JKLUxl6iPD1EGi0gTM4VGlnvibwBl5Vqv00BYn-yxTkH010T-hkIV2R7AbxJiaY2Q4LAdzg07_zjXXd0ilcIV6YeOlWyZ5I9tNMZNWsVzj7siML7owpdH5dvxWNW4LktnWB2lNGZX2_BzaOvkfA9Y0XpJ=w993-h252-no?authuser=0)

# Backups

6. How to export and backup PEN's XWC account private key if you forget to back up the private key when running PEN for the first time

`./pen dumpkey --data-dir=/home/pen/pen_data`

[https://lh3.googleusercontent.com/pw/AM-JKLVh_47pEIn9t0OKbUgXVDMgFlhvG3xLbd2v5Zev8iUf1OPThNVX561tcUF-h-NUWlKgmj_kjzxDGFHALKuVFgoKPdrr-EByrEgv3LMKM24UVmCXOLqmxSkFGOtP19T2Ek0ImCUxXE4RoZcuLFS15E9F=w993-h140-no?authuser=0](https://lh3.googleusercontent.com/pw/AM-JKLVh_47pEIn9t0OKbUgXVDMgFlhvG3xLbd2v5Zev8iUf1OPThNVX561tcUF-h-NUWlKgmj_kjzxDGFHALKuVFgoKPdrr-EByrEgv3LMKM24UVmCXOLqmxSkFGOtP19T2Ek0ImCUxXE4RoZcuLFS15E9F=w993-h140-no?authuser=0)

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
