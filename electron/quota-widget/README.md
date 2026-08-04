# TokenFactory 额度悬浮窗

独立于 TokenFactory 主桌面窗口的 Windows 极简额度查询工具。

## 开发运行

```powershell
cd C:\work\token-factory\electron
npm install
npm run start:quota
```

首次运行可选择 `tokease.com` 国际站、`tokease.cn` 国内站或自定义地址，然后填写 API Key。API Key 和站点地址只保存在本次运行的主进程内存中，不写入磁盘。

应用、托盘和 Windows 可执行文件使用 `https://tokease.com/logo.png` 对应的本地 Logo 资源，运行时不依赖远程图片。

## 测试

```powershell
npm run test:quota
```

## Windows 便携版

```powershell
npm run build:quota:win
```

生成文件位于 `electron/dist-quota/`。
