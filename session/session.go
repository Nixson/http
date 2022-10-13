package session

import (
	"encoding/json"
	"github.com/Nixson/logger"
	"os"
	"sync"
	"time"
)

type Session struct {
	User
	Hash  string `json:"hash"`
	Dtime int64  `json:"dtime"`
}

func (s *Session) Set(user User) {
	s.Id = user.Id
	s.Login = user.Login
	s.Username = user.Username
	s.Access = user.Access
	CreateFullSession(s, false)
}

const Dtime = 3600

var Sessions sync.Map

const sessionPath = "./bin/session"

func init() {
	nowTime := time.Now().Unix()
	logger.Println("init Session")
	files, _ := os.ReadDir(sessionPath)
	for _, f := range files {
		info, _ := f.Info()
		if info.ModTime().Unix() > nowTime-Dtime {
			byteValue, _ := os.ReadFile(sessionPath + "/" + f.Name())
			var sess Session
			_ = json.Unmarshal(byteValue, &sess)
			CreateFullSession(&sess, true)
		} else {
			_ = os.Remove(sessionPath + "/" + f.Name())
		}
	}
}

func CreateSession(sess *Session) *Session {
	return CreateFullSession(sess, true)
}

func CreateFullSession(sess *Session, ignoreFile bool) *Session {
	sess.Dtime = time.Now().Unix() + Dtime
	Sessions.Store(sess.Hash, sess)
	if !ignoreFile {
		data, _ := json.Marshal(sess)
		_ = os.WriteFile(sessionPath+"/"+sess.Hash+".json", data, 0777)
	}
	var sub = *sess
	return &sub
}

func GetSession(hash string) *Session {
	if sess_, found := Sessions.Load(hash); found {
		runtime := time.Now().Unix()
		sess := sess_.(*Session)
		if runtime < sess.Dtime {
			sess.Dtime = time.Now().Unix() + Dtime
			return sess
		}
		DeleteSession(hash)
	}
	return nil
}
func DeleteSession(hash string) {
	Sessions.Delete(hash)
	_ = os.Remove(sessionPath + "/" + hash + ".json")
}
