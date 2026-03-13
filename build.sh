#!/bin/bash

# 项目名称
APP_NAME="nebula-server-watcher"
# 输出目录
BUILD_DIR="_bin"

# 检查 upx 是否可用
UPX_AVAILABLE=false
if command -v upx >/dev/null 2>&1; then
    UPX_AVAILABLE=true
    echo "UPX found, will compress binaries."
else
    echo "Warning: UPX not found, skipping compression."
fi

# 目标平台列表: "OS/ARCH"
PLATFORMS=(
    "linux/amd64"
    # "linux/arm64"
    # "darwin/amd64"
    # "darwin/arm64"
    # "windows/amd64"
)

# 创建输出目录
mkdir -p $BUILD_DIR

echo "Starting cross-platform build..."

for PLATFORM in "${PLATFORMS[@]}"; do
    # 分割 OS 和 ARCH
    IFS="/" read -r -a PART <<< "$PLATFORM"
    GOOS=${PART[0]}
    GOARCH=${PART[1]}
    
    # 设置输出文件名
    OUTPUT_NAME="${APP_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    FILE_PATH="${BUILD_DIR}/${OUTPUT_NAME}"
    echo "Building for ${GOOS}/${GOARCH}..."
    
    # 执行编译
    # -ldflags="-s -w" 移除符号表和调试信息
    # -trimpath 移除源码路径
    env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags="-s -w" -o "${FILE_PATH}" .
    
    if [ $? -eq 0 ]; then
        # 如果 upx 可用且不是 macOS ARM64 (upx 对此支持可能不稳定)
        if [ "$UPX_AVAILABLE" = true ]; then
            echo "Compressing ${OUTPUT_NAME} with UPX..."
            upx --best "${FILE_PATH}" >/dev/null 2>&1
        fi
    else
        echo "Error: Build failed for ${PLATFORM}"
    fi
done

echo "----------------------------------------"
echo "Build complete! Artifacts are in ${BUILD_DIR}/"
ls -lh $BUILD_DIR
