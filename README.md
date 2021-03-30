# lwe_proto
A Light-Weight and Extensible(LWE) binary protocol compiler framework 

# Why need lwe_proto
As a C/go developer sometimes I need use binary format (JSON is not space efficient for some case) to exchange message between server and client SDK, So I write this tool to ease the process of:
> 1. Define the **message ID** and **message structure**
> 2. Write the **Encode/Decode** source code for the messages

With **lwe_proto** We just care about the message ID and layout, whenever We want add new messages or change the message format, We just need change the definition file, then use **lwe_proto** to generate the  **Encode/Decode** logic

In my work, I write the server in golang, and SDK in C for Android/iOS/Windows platform, for security reasons, Here only open source the golang generator. It is easy to write generator for other languages.

# Why not proto buffer
Proto buffer is a powerful protocol format compiler, You should first consider use it unless:
> 1. You need fullly control the message bianry layout
> 2. You want keep the project dependency as less as possible

# Example
proto file content:
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

Generate codec code in go

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

# Features
1. Support uint8, uint16, uint32, and uint64 types
2. Support simple custom error checks
3. Support variable length byte array
4. Custom bind message id to message structure

# How it works
Basically it works like a language interpreter with below process:
> 1. lexical analysis
> 2. syntax analysis
> 3. semantic analysis, generate the Abstruct Syntax Tree
> 4. interprete the AST. 
For **lwe_proto** It do the main work in step 4: walking the AST and generate the message codes
