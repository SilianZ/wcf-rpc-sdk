# WeChatFerry Docker

本项目基于 Fedora，使用 LXDE, Wine, xRDP, WeChatFerry 构建 WeChat 的 Docker 镜像。

## 功能

- 通过 xRDP 远程访问桌面环境。
- 使用 WeChatFerry 注入 SDK 到 WeChat 进程。
- 提供 WeChatFerry 的命令行和消息接口。

## 构建

本项目使用 Github Action 自动构建 Docker 镜像。

### 手动构建

1. 克隆仓库：
   ```bash
   git clone <your_repo_url>
   cd <your_repo_name>
   ```

2. 构建镜像：
   ```bash
   docker build -t wechat_ferry:latest -f docker/Dockerfile .
   ```

## 使用 Docker Compose 运行

1. 准备微信数据目录 (可选)：
   ```bash
   mkdir -p ./wechat/program
   mkdir -p ./wechat/share/icons
   mkdir -p ./wechat/user_dat
   ```

2. 使用 `docker-compose` 启动容器：
   ```bash
   docker-compose up -d
   ```

## 使用

1. 通过 xRDP 客户端连接到 `localhost:13389`，使用用户名 `root` 和密码 `123` 登录。
2. 双击桌面上的 `WeChatFerry` 图标启动微信。

## 注意

- 首次运行需要先安装微信，双击桌面上的 `WeChatSetup` 图标进行安装。
- 微信的程序文件和用户数据保存在宿主机的 `./wechat` 目录下。

## 贡献

欢迎提交 Issue 和 Pull Request。

## 许可证

本项目基于 MIT 许可证。

