package service

import (
	"fmt"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"net"
	"time"
)

type ServerCmdService struct{}

// List
func (is *ServerCmdService) List(page, pageSize uint) (res *model.ServerCmdList) {
	res = &model.ServerCmdList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.ServerCmd{})
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.ServerCmds)
	return
}

// Info
func (is *ServerCmdService) Info(id uint) *model.ServerCmd {
	u := &model.ServerCmd{}
	DB.Where("id = ?", id).First(u)
	return u
}

// Delete
func (is *ServerCmdService) Delete(u *model.ServerCmd) error {
	return DB.Delete(u).Error
}

// Create
func (is *ServerCmdService) Create(u *model.ServerCmd) error {
	res := DB.Create(u).Error
	return res
}

// SendCmd 发送命令
func (is *ServerCmdService) SendCmd(port int, cmd string, arg string) (string, error) {
	//组装命令
	cmd = cmd + " " + arg
	res, err := is.SendSocketCmd("v6", port, cmd)
	if err == nil {
		return res, nil
	}
	//v6连接失败，尝试v4
	res, err = is.SendSocketCmd("v4", port, cmd)
	if err == nil {
		return res, nil
	}
	return "", err
}

// SendSocketCmd
func (is *ServerCmdService) SendSocketCmd(ty string, port int, cmd string) (string, error) {
	addr := "[::1]"
	tcp := "tcp6"
	if ty == "v4" {
		tcp = "tcp"
		addr = "127.0.0.1"
	}
	conn, err := net.DialTimeout(tcp, fmt.Sprintf("%s:%v", addr, port), 2*time.Second)
	if err != nil {
		Logger.Debugf("%s connect to id server failed: %v", ty, err)
		return "", err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	// 发送命令
	_, err = conn.Write([]byte(cmd))
	if err != nil {
		Logger.Debugf("%s send cmd failed: %v", ty, err)
		return "", err
	}

	// 循环读取完整返回
	var result []byte
	buf := make([]byte, 4096)
	for {
		n, rerr := conn.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if rerr != nil {
			break
		}
	}
	return string(result), nil
}

func (is *ServerCmdService) Update(f *model.ServerCmd) error {
	return DB.Model(f).Updates(f).Error
}
