package netutil

import (
	"errors"
	"net"
	"os"
	"regexp"
	"strings"
)

// ignoreInterfacePattern 匹配需忽略的虚拟/环回/容器桥接网卡
// 避免在 --net=host 或宿主机下误取 docker0、veth、bridge 等无效 IP
var ignoreInterfacePattern = regexp.MustCompile(`^(docker\d+|br-.*|veth.*|tun\d+|tap\d+|wg\d+|lo)$`)

// ResolveRegisterIP 智能推导服务注册所需的本机/容器 IP
//
// 解析策略（阶梯兜底）：
// 1. 优先读取常见容器与编排平台环境变量 (HOST_IP / POD_IP / REGISTER_IP)
// 2. 若传入了注册中心地址（如 Nacos 地址），利用系统路由表通过 UDP 出网嗅探判定出口 IP
// 3. 遍历本地网卡，过滤掉 docker0/veth/lo 等虚拟网卡，选取首个有效的内网 IPv4
//
// 参数：
//   - targetAddr: 目标注册中心地址（如 "127.0.0.1:8848"），用于路由出口判定，可为空
//
// 返回：
//   - string: 解析出的有效 IPv4 字符串
//   - error: 探测失败时的错误信息
func ResolveRegisterIP(targetAddr string) (string, error) {
	// 1. 优先检测环境变量（解决 Docker NAT 端口映射与 Kubernetes Pod IP 注入）
	for _, envKey := range []string{"HOST_IP", "POD_IP", "REGISTER_IP"} {
		if ip := strings.TrimSpace(os.Getenv(envKey)); ip != "" && IsValidIPv4(ip) {
			return ip, nil
		}
	}

	// 2. 通过连接目标注册中心，利用系统内核路由表裁决出口网卡
	// 注意：net.Dial("udp", ...) 不会进行 TCP 三次握手或实际向远端发包，仅绑定本地出网套接字
	if targetAddr != "" {
		if ip, err := getOutboundIP(targetAddr); err == nil && IsValidIPv4(ip) {
			return ip, nil
		}
	}

	// 3. 兜底方案：枚举物理网卡并过滤无效虚拟网卡
	return getLocalIPByInterfaces()
}

// getOutboundIP 通过向目标地址建立伪 UDP 连接获取本端出口 IP
//
// 参数：
//   - targetAddr: 目标地址，如 "192.168.1.10:8848"
//
// 返回：
//   - string: 本地出口 IP
//   - error: 套接字创建失败错误
func getOutboundIP(targetAddr string) (string, error) {
	// 如果传入 targetAddr 缺少端口，补充一个假端口以满足 net.Dial 的解析要求
	if !strings.Contains(targetAddr, ":") {
		targetAddr = targetAddr + ":80"
	}

	conn, err := net.Dial("udp", targetAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", errors.New("cannot cast local address to *net.UDPAddr")
	}

	return localAddr.IP.String(), nil
}

// getLocalIPByInterfaces 遍历本地网卡设备并提取首个合法 IPv4
//
// 返回：
//   - string: 本地单播 IPv4 地址
//   - error: 未找到合适 IP 时返回错误
func getLocalIPByInterfaces() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// 忽略处于 DOWN 状态或 Loopback 回环网卡
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 正则黑名单过滤（排除 docker0, br-*, veth*, tun* 等桥接与虚拟网络设备）
		if ignoreInterfacePattern.MatchString(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				// 仅接受有效且非环回的 IPv4
				if ip4 := ipNet.IP.To4(); ip4 != nil && !ipNet.IP.IsLoopback() {
					return ip4.String(), nil
				}
			}
		}
	}

	return "", errors.New("no valid non-loopback IPv4 interface found")
}

// IsValidIPv4 校验 IP 字符串是否为非未指定(0.0.0.0)、非回环的合法 IPv4 地址
//
// 参数：
//   - ipStr: 待校验的 IP 字符串
//
// 返回：
//   - bool: 是否合法
func IsValidIPv4(ipStr string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	// 仅允许 IPv4，且排除 0.0.0.0 与 127.0.0.1
	return ip.To4() != nil && !ip.IsUnspecified() && !ip.IsLoopback()
}
