package wshelper

import (
	"errors"
	"net/http"
	"time"

	"io"

	"io/ioutil"

	"github.com/gorilla/websocket"
)

const (
	//TextMessage 定义消息类型为字符串
	TextMessage = 1
	//StreamMessage 定义消息类型为数据流
	StreamMessage = 2

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage = 8

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage = 9

	// PongMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage = 10
)

//WebSocketHelper 用来升级http协议到ws协议
type WebSocketHelper struct {
	AuthHandleFunc      func(r *http.Request) bool                    //验证访问是否合法
	CloseHandleFunc     func(closeStatus int, closeText string) error //当连接关闭时调用此方法
	KeepAlive           bool                                          //是否保持连接存活,默认不保持
	TextMsgHandleFunc   func(msg string) error                        //当收到文本消息时调用此方法
	StreamMsgHandleFunc func(r io.Reader) error                       //当收到字节流消息时调用此方法
	PingHandleFunc      func(pingMsg string) error                    //当收到ping消息时调用此方法
	PongHandleFunc      func(pongMsg string) error                    //当收到pong消息时调用此方法
	c                   *websocket.Conn
	isAlive             chan bool
	rbufSize            int
	wbufSize            int
	writer              *io.Writer
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
							h.Close()
							return
						}
					case <-t.C:
						h.Close()
						return
					}
				}
			}
		}()
	}
	for {

		msgType, r, err := h.c.NextReader()
		if err != nil {
			return errors.New("recive message fail, err:" + err.Error())
		}
		switch msgType {
		case TextMessage:
			msg, err := ioutil.ReadAll(r)
			if err != nil {
				return err
			}
			if string(msg) == "control:ping" { //这里兼容前端js，"control:ping"将被认为是ping消息
				h.WriteMessage(msgType, []byte("control:pong"))
			} else if string(msg) == "control:pong" { //这里兼容前端js，"control:pong"将被认为是pong消息
				h.isAlive <- true
			} else {
				err := h.TextMsgHandleFunc(string(msg))
				if err != nil {
					return errors.New("fail to write data")
				}
			}
		case StreamMessage:
			err := h.StreamMsgHandleFunc(r)
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
//message 指定要写入的数据：string 或 io.Reader
func (h *WebSocketHelper) WriteMessage(messageType int, message interface{}) error {
	switch value := message.(type) {
	case string:
		return h.c.WriteMessage(messageType, []byte(value))
	case io.Reader:
		_, err := io.Copy(h.c.UnderlyingConn(), value)
		return err
	default:
		return errors.New("unkown data type")
	}
}

// WriteControl 按照给定的超时，写入控制信息
// 允许的控制信息类型有： CloseMessage, PingMessage 以及 PongMessage.
func (h *WebSocketHelper) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return h.c.WriteControl(messageType, data, deadline)
}
