# 对比说明

> 英文主文档：[comparison.md](comparison.md)

## 一览

| 项目 | 语言 | 角色 | 配置模型 | 适合 |
|------|------|------|----------|------|
| **sipplane** | Go | 信令面（代理/注册/边缘） | 声明式资源 + 控制面 | 云原生 Go、K8s、策略热更新 |
| Kamailio | C | SIP 代理/服务器 | 本地脚本 + 模块 | 极致 CPS、IMS、已有大规模部署 |
| OpenSIPS | C | SIP 代理（偏平台） | 脚本 + 丰富内建能力 | 集成 B2BUA、负载均衡、运维工具 |
| sipgo | Go | SIP **库** | 代码 | 用 Go 写任意 SIP 服务 |
| LiveKit SIP | Go | SIP ↔ WebRTC | LiveKit API | 电话进 LiveKit Room |

## 与 Kamailio / OpenSIPS

学习其**行为与能力清单**，不克隆脚本方言，不以「单机 CPS 第一」为唯一 KPI。
差异化在：共享状态水平扩展、声明式/GitOps、Go 云原生运维体验、信令与媒体清晰分离。

## 与 sipgo

sipplane **依赖 sipgo**，是产品化平台，不是协议栈分叉。栈层 bug 优先回馈上游。

## 一句话

> **sipgo** 处理 SIP 消息；**sipplane** 把 SIP 边缘当成云服务来运营。
> Kamailio/OpenSIPS 仍是成熟 C 代理金标准 —— sipplane 是 Go 原生的另一套架构，不是二进制级克隆。
