# 使用 Fedora 作为基础镜像
FROM fedora:40 AS builder

LABEL describe="Fedora for WeChat, base on LXDE, Wine, xRDP, WeChatFerry."

WORKDIR /root

# 安装基础运行环境和 Go 开发环境
RUN dnf groupinstall -y "LXDE" \
        && dnf install -y wine \
        && dnf install -y xrdp \
        && dnf clean all

# 安装编译工具
RUN dnf install -y go \
        && dnf clean all

# 复制注入器源码
COPY ./injector /root/injector

# 编译注入器
RUN cd /root/injector \
        && GOOS=windows go build -o injector.exe \
        && cd /root \
        && rm -rf /root/injector \
        && rm -rf .cache/go-build

# 第二阶段：构建运行环境
FROM fedora:40

WORKDIR /root

# 从构建阶段复制编译好的注入器
COPY --from=builder /root/injector.exe /root/injector.exe

# 安装基础运行环境
RUN dnf groupinstall -y "LXDE" \
        && dnf install -y wine \
        && dnf install -y xrdp \
        && dnf clean all

# 复制资料文件
COPY ./res ./res

# 设置环境变量
ARG WECHAT_URL=https://github.com/lich0821/WeChatFerry/releases/download/v39.3.5/WeChatSetup-3.9.11.25.exe
ARG SDK_URL=https://github.com/lich0821/WeChatFerry/releases/download/v39.3.5/v39.3.5.zip

# 下载微信安装包和 SDK
RUN curl -o /root/WeChatSetup.exe ${WECHAT_URL} \
        && curl -o /root/res/sdk.zip ${SDK_URL} \
        && cd /root/res \
        && unzip sdk.zip \
        && rm sdk.zip

# 部署运行环境
RUN echo "root:123" | chpasswd \
        && mkdir ~/Desktop \
        && mv res/*.desktop ~/Desktop

# Port for xRDP
EXPOSE 3389
# Port for WeChatFerry command
EXPOSE 8001
# Port for WeChatFerry message
EXPOSE 8002

ENTRYPOINT ["./res/entrypoint.sh"]
CMD ["bash"]

