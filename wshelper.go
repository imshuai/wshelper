package wshelper

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

//WebSocketHelper 用来升级http协议到ws协议
type WebSocketHelper struct {
	AuthHandleFunc    func(r *http.Request) bool                         //验证访问是否合法
	CloseHandleFunc   func(closeStatus int, closeText string) error      //当连接关闭时调用此方法
	KeepAlive         bool                                               //是否保持连接存活,默认不保持
	MessageHandleFunc func(messageType int, msg []byte, err error) error //当收到消息时吊用此方法，提供消息类型messageType和消息字节数组msg
	PingHandleFunc    func(pingMsg string) error                         //当收到ping消息时调用此方法
	PongHandleFunc    func(pongMsg string) error                         //当收到pong消息时吊用此方法
	c                 *websocket.Conn
	isAlive           chan bool
	rbufSize          int
	wbufSize          int
}

//NewWebSocketHelper 创建新的请求升级器
//rbufSize 指定读缓存大小，字节
//wbufSize 指定写缓存大小，字节
func NewWebSocketHelper(rbufSize, wbufSize int) *WebSocketHelper {
	return &WebSocketHelper{
		rbufSize:  rbufSize,
		wbufSize:  wbufSize,
		KeepAlive: false,
	}
}

func (h *WebSocketHelper) upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	u := websocket.Upgrader{
		ReadBufferSize:  h.rbufSize,
		WriteBufferSize: h.wbufSize,
	}
	if h.AuthHandleFunc != nil {
		u.CheckOrigin = h.AuthHandleFunc
	}
	return u.Upgrade(w, r, nil)
}

//StartHandle 开始介入管理websocket连接
//w http请求的写入流
//r http请求的指针
func (h *WebSocketHelper) StartHandle(w http.ResponseWriter, r *http.Request) error {
	var err error
	h.c, err = h.upgrade(w, r)
	if err != nil {
		return err
	}
	defer h.Close()

	if h.CloseHandleFunc != nil {
		h.c.SetCloseHandler(h.CloseHandleFunc)
	}
	if h.PingHandleFunc != nil {
		h.c.SetPingHandler(h.PingHandleFunc)
	}
	if h.PongHandleFunc != nil {
		h.c.SetPongHandler(h.PongHandleFunc)
	}
	if h.KeepAlive {
		h.isAlive = make(chan bool)
		go func() {
			t := time.NewTicker(time.Second * 10)
			for {
				select {
				case <-t.C:
					// select {
					// case <-isAlive: //这里判断chan中是否有数据，保证之后读取的时候是空的
					// case <-time.NewTicker(time.Millisecond * 100).C:
					// }
					h.c.WriteMessage(websocket.TextMessage, []byte("control:ping"))
					select {
					case i := <-h.isAlive:
						if !i {
							log.Println("user is not alive, closing the connection")
							h.Close()
							return
						}
					case <-t.C:
						log.Println("timeout, user is not alive, closing the connection")
						h.Close()
						return
					}
				}
			}
		}()
	}
	for {
		messageType, msg, err := h.c.ReadMessage()
		if err != nil {
			log.Println("recive message fail, err:", err)
			return errors.New("recive message fail, err:" + err.Error())
		}
		switch messageType {
		case 1:
			if string(msg) == "control:ping" { //这里兼容前端js，"control:ping"将被认为是ping消息
				h.WriteMessage(messageType, []byte("control:pong"))
			} else if string(msg) == "control:pong" { //这里兼容前端js，"control:pong"将被认为是pong消息
				h.isAlive <- true
			} else {
				log.Println("recived a text msg")
				err := h.MessageHandleFunc(messageType, msg, err)
				if err != nil {
					return errors.New("fail to write data")
				}
			}
		case 2:
			log.Println("recived a binary msg")
			err := h.MessageHandleFunc(messageType, msg, err)
			if err != nil {
				return errors.New("fail to write data")
			}
		}
	}
}

//Close 关闭WebSocketHelper的websocket连接
func (h *WebSocketHelper) Close() error {
	return h.c.Close()
}

//WriteMessage 向websocket连接写入数据
//messageType 指定写入数据的类型1代表文本数据，2代表二进制数据
//message 指定要写入数据的字节数组
func (h *WebSocketHelper) WriteMessage(messageType int, message []byte) error {
	return h.c.WriteMessage(messageType, message)
}

//ReadMessage 从websocket连接读出数据
//返回值：
//messageType 指示读出数据的类型1代表文本数据，2代表二进制数据
//message 指示读出数据的字节数组
//err 指示读取数据是产生的错误
func (h *WebSocketHelper) ReadMessage() (messageType int, message []byte, err error) {
	messageType, message, err = h.c.ReadMessage()
	return messageType, message, err
}

// WriteControl writes a control message with the given deadline.
// The allowed message types are CloseMessage, PingMessage and PongMessage.
func (h *WebSocketHelper) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return h.c.WriteControl(messageType, data, deadline)
}
