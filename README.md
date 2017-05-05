# WSHelper
Helper for package `gorilla/websocket` 
基于`github.com/gorilla/websocket`包二次开发的工具组件  

- 事件回调机制  
  1. `AuthHandleFunc`定义访问认证事件回调函数  
     直接传递给`gorilla/websocket`的`Upgrader.CheckOrigin`命令
  2. `CloseHandleFunc`定义连接关闭事件回调函数  
     直接传递给`gorilla/websocket`的`Conn.SetCloseHandler`命令  
  3. `PingHandleFunc`定义接收到ping消息事件回调函数  
     直接传递给`gorilla/websocket`的`Conn.SetPingHandler`命令  
  4. `PongHandleFunc`定义接收到pong消息事件回调函数  
     直接传递给`gorilla/websocket`的`Conn.SetPongHandler`命令 
  5. `MessageHandleFunc`定义接收到消息事件回调函数        
- 自定义ping消息以兼容前端js无原生ping命令问题[TODO]  

