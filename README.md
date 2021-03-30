# lwe_proto
A Light-Weight and Extensible(LWE) network binary protocol compiler framework 

# why not proto buffer
Proto buffer is a powerful protocol format compiler, You should first consider use it unless:
> 1. You need fullly control the message bianry layout
> 2. You want keep the project dependency as less as possible

# why need lwe_proto
As a C/go developer sometimes I need use binary format (JSON is not space efficient for some case) to exchange message between server and client SDK, So I write this tool to ease the process of:
> 1. Define the **message ID** and **message structure**
> 2. Write the **Encode/Decode** source code for the messages

With **lwe_proto** We just care about the message ID and layout, whenever We want add new messages or change the message format, We just need change the definition file, then use **lwe_proto** to generate the  **Encode/Decode** logic

In my work, I write the server in golang, and SDK in C for Android/iOS/Windows platform, for security reasons, Here only open source the golang generator. It is easy to write generator for other languages.

# examples

# how it works
Basically it works like a language interpreter with below process:
> 1. lexical analysis
> 2. syntax analysis
> 3. semantic analysis, generate the Abstruct Syntax Tree
> 4. interprete the AST. 
For **lwe_proto** It do the main work in step 4: walking the AST and generate the message codes
