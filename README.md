[English Version](README-en.md)

# lwe_proto
一个轻量可扩展(Light-Weight, Extensible)的二进制协议序列化编译工具框架, 功能类似于类似与proto buffer, 但试图做的更轻量易懂并减少依赖.

# 为什么需要lwe_proto
作为一个C/go程序员, 我需要在服务器和客户端之间用二进制的消息协议来交换数据(JSON不够经济), 基本过程大概是这样:
> 1. 定义消息ID和消息体内容字段
> 2. 编写消息体的编解码代码, 然后编写根据消息ID解消息内容的代码
迭代时, 需要重复步骤1-2, 而步骤2重复性很高而且容易出错

为了简化这样的开发流程, 我编写了**lwe_proto**, 有了这个工具之后, 流程变成:
> 1. 定义消息ID和消息体内容字段
> 2. 使用**lwe_proto**来生成消息的编解码代码

使用**lwe_proto**之后, 我们只需要关心消息ID和消息结构的定义, 当需要新增或者修改协议消息时, 只需要修改消息定义文件, 然后重新生成就哦了, 大大简化了开发的流程并减少出错的可能性.

在我的工作中, 服务是用go写的, 客户端的SDK是用C写, 因此工具可以生成go和C的代码并且可以相互工作, 出于保密原因, 这里只开放go的生成部分, 你根据自己的需要对**lwe_proto**进行扩展, 有生成go的部分做参考, 应该不是一件很难的事情.

# 为什么不使用proto buffer
proto buffer是一个强大的协议序列化生成工具, 你应该首先考虑用它, 除非你有下面的需求:
> 1. 你需要对生成的代码有完全的控制, 或者需要自定义一些内容
> 2. 你想自己项目尽可能少的依赖外部函数库

# 使用例子(ubuntu环境下)

1. 查看协议文件内容:
```bash
$cat data/test.proto 

//use // for normal comment in proto files
//use //* for comment that need to write to source code

//mspace define the message space
mspace lwe

//define const
const ProtoVersion           0x01

defmid lwe_msgid {
    //*base comment
    Lwe_msg_base = 0,
    Lwe_msg_connect,
    Lwe_msg_connect_ack,
}

//define message bind 
bind Lwe_msg_connect            LweMsg_Connect
bind Lwe_msg_connect_ack        nil //bind to nil means this message id has no body

//define 2-byte header
defmsg LweMsg_Header {
    Version         u2 -> equal ProtoVersion //this will be checked by generated code
    Flags           u6
    MessageId       u8
}

const MaxNameSize     20

defmsg LweMsg_Connect {
    IP              u32
    Port            u16
    NameLen         u8  ->  max MaxNameSize
    Name            []u8 -> limit by NameLen
}

```

2. 生成go代码
```bash
$go build .
$./lwe_proto -f data/test.proto 
```
```go
/*
 * code auto generated from: data/test.proto @2021-03-29 19:56:12, Do NOT touch by hand!!!
 * generator version 1.0; author lqp; mode: golang
*/
const ProtoVersion = 1 //0x1
const (
    //base comment
    Lwe_msg_base = 0 //hex: 0x0
    Lwe_msg_connect = 1 //hex: 0x1
    Lwe_msg_connect_ack = 2 //hex: 0x2
)

func lwe_msgid_name(id uint16)(string, bool) {
    switch id {
    case Lwe_msg_base:
        return "Lwe_msg_base", true
    
    case Lwe_msg_connect:
        return "Lwe_msg_connect", true
    
    case Lwe_msg_connect_ack:
        return "Lwe_msg_connect_ack", true
    }
    return "", false
}

type LweMsg_Header struct {
    Version uint8 //u2
    Flags uint8 //u6
    MessageId uint8 //u8
}

func encode_LweMsg_Header(buf io.Writer, m *LweMsg_Header) int {
    tmp := uint8(0)
    tmp |= (m.Version & 0x3) << 6
    tmp |= m.Flags & 0x3f
    if binary.Write(buf, binary.BigEndian, tmp) != nil { return -1 }
    
    if binary.Write(buf, binary.BigEndian, m.MessageId) != nil { return -1 }
    return 0
}

func decode_LweMsg_Header(buf io.Reader, m *LweMsg_Header) int {
    tmp := uint8(0)
    if binary.Read(buf, binary.BigEndian, &tmp) != nil { return -1 }
    m.Version = (tmp >> 6) & 0x3
    if m.Version != ProtoVersion { return -1 }
    m.Flags = tmp & 0x3f;
    if binary.Read(buf, binary.BigEndian, &m.MessageId) != nil { return -1 }
    return 0
}
const MaxNameSize = 20 //0x14

type LweMsg_Connect struct {
    IP uint32 //u32
    Port uint16 //u16
    NameLen uint8 //u8
    Name [MaxNameSize]uint8
}

func encode_LweMsg_Connect(buf io.Writer, m *LweMsg_Connect) int {
    if binary.Write(buf, binary.BigEndian, m.IP) != nil { return -1 }
    if binary.Write(buf, binary.BigEndian, m.Port) != nil { return -1 }
    
    if m.NameLen > MaxNameSize { m.NameLen = MaxNameSize} 
    if binary.Write(buf, binary.BigEndian, m.NameLen) != nil { return -1 }
    
    if binary.Write(buf, binary.BigEndian, m.Name[0:m.NameLen]) != nil { return -1 }
    return 0
}

func decode_LweMsg_Connect(buf io.Reader, m *LweMsg_Connect) int {
    if binary.Read(buf, binary.BigEndian, &m.IP) != nil { return -1 }
    if binary.Read(buf, binary.BigEndian, &m.Port) != nil { return -1 }
    if binary.Read(buf, binary.BigEndian, &m.NameLen) != nil { return -1 }
    if m.NameLen > MaxNameSize { return -1 }
    if binary.Read(buf, binary.BigEndian, m.Name[:m.NameLen]) != nil { return -1 }
    return 0
}

func encodeLweMsgById(buf io.Writer, mid uint16, msg interface{}) int {
    switch mid {
    case Lwe_msg_connect:
        return encode_LweMsg_Connect(buf, msg.(*LweMsg_Connect))
    
    case Lwe_msg_connect_ack:
        return 0
    }
    
    return -1
}

func decodeLweMsgById(buf io.Reader, mid uint16, msg interface{}) int {
    switch mid {
    case Lwe_msg_connect:
        return decode_LweMsg_Connect(buf, msg.(*LweMsg_Connect))
    
    case Lwe_msg_connect_ack:
        return 0
    }
    
    return -1
}

```

# 特性
1. 支持uint8, uint16, uint32, and uint64类型
2. 支持比特字段, 例如2bit,3-bit的字段
3. 支持变长字节数组
4. 支持简单的编解码错误判断
5. 自定义消息ID和消息体的绑定

# 它是如何工作的
它的工作方式和语言解释器类似, 主要包括以下几个步骤:
> 1. 词法分析
> 2. 语法分析
> 3. 语义分析, 生成抽象语法树AST
> 4. 解释执行AST
对于 **lwe_proto** 来说, 它的主要工作在第4步, 遍历AST来生成相应的代码, 这里也是扩展其他生成语言唯一需要修改的地方.
