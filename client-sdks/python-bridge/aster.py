"""
Aster Bridge - Python SDK for Programmatic Tool Calling

This module provides async functions to call Aster tools from Python code.
It's designed to be injected into code execution environments automatically.

Usage:
    # Automatically injected by RuntimeManager
    content = await Read(path="file.txt")
    await Write(path="output.txt", content=content.upper())
"""

import aiohttp
import asyncio
import os
import time
from typing import Any, Dict, Optional


class AsterBridgeError(Exception):
    """Aster 桥接错误基类"""
    pass


class ToolExecutionError(AsterBridgeError):
    """工具执行错误"""
    def __init__(self, tool_name: str, error_msg: str):
        self.tool_name = tool_name
        self.error_msg = error_msg
        super().__init__(f"Tool {tool_name} failed: {error_msg}")


class NetworkError(AsterBridgeError):
    """网络错误"""
    pass


class AsterBridge:
    """
    Aster 工具桥接客户端

    通过 HTTP API 调用 Go 侧的 Aster 工具
    """

    def __init__(
        self,
        base_url: Optional[str] = None,
        max_retries: int = 3,
        retry_delay: float = 0.5,
    ):
        """
        初始化桥接客户端

        Args:
            base_url: HTTP 桥接服务器地址,默认从环境变量 ASTER_BRIDGE_URL 获取
            max_retries: 最大重试次数(默认3次)
            retry_delay: 重试延迟秒数(默认0.5秒,指数退避)
        """
        self.base_url = base_url or os.environ.get(
            "ASTER_BRIDGE_URL", "http://localhost:8080"
        )
        self.max_retries = max_retries
        self.retry_delay = retry_delay
        self._session: Optional[aiohttp.ClientSession] = None

    async def _get_session(self) -> aiohttp.ClientSession:
        """获取或创建 HTTP 会话"""
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession()
        return self._session

    async def call_tool(self, name: str, **kwargs) -> Any:
        """
        调用工具(带重试逻辑)

        Args:
            name: 工具名称
            **kwargs: 工具输入参数

        Returns:
            工具执行结果

        Raises:
            ToolExecutionError: 工具执行失败
            NetworkError: 网络错误(重试后仍失败)
        """
        last_error = None

        for attempt in range(self.max_retries):
            try:
                session = await self._get_session()

                async with session.post(
                    f"{self.base_url}/tools/call",
                    json={"tool": name, "input": kwargs},
                    timeout=aiohttp.ClientTimeout(total=60),
                ) as resp:
                    # 检查 HTTP 状态码
                    if resp.status >= 500:
                        # 服务器错误,可以重试
                        error_text = await resp.text()
                        last_error = NetworkError(
                            f"Server error (HTTP {resp.status}): {error_text}"
                        )
                        if attempt < self.max_retries - 1:
                            await asyncio.sleep(self.retry_delay * (2 ** attempt))
                            continue
                        raise last_error

                    if resp.status >= 400:
                        # 客户端错误,不重试
                        error_text = await resp.text()
                        raise NetworkError(
                            f"Client error (HTTP {resp.status}): {error_text}"
                        )

                    # 解析响应
                    result = await resp.json()

                    # 检查工具执行结果
                    if not result.get("success"):
                        error_msg = result.get("error", "Unknown error")
                        raise ToolExecutionError(name, error_msg)

                    return result.get("result")

            except aiohttp.ClientConnectorError as e:
                # 连接错误,可以重试
                last_error = NetworkError(
                    f"Connection error: {str(e)}. "
                    f"Is the bridge server running at {self.base_url}?"
                )
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay * (2 ** attempt))
                    continue
                raise last_error

            except aiohttp.ClientError as e:
                # 其他网络错误,可以重试
                last_error = NetworkError(f"Network error: {str(e)}")
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay * (2 ** attempt))
                    continue
                raise last_error

            except ToolExecutionError:
                # 工具执行错误,不重试,直接抛出
                raise

            except asyncio.TimeoutError:
                # 超时错误,可以重试
                last_error = NetworkError(f"Tool {name} timed out after 60 seconds")
                if attempt < self.max_retries - 1:
                    await asyncio.sleep(self.retry_delay * (2 ** attempt))
                    continue
                raise last_error

        # 所有重试都失败
        if last_error:
            raise last_error
        raise NetworkError(f"Failed to call tool {name} after {self.max_retries} attempts")

    async def list_tools(self) -> list:
        """
        列出所有可用工具

        Returns:
            工具名称列表
        """
        session = await self._get_session()

        async with session.get(f"{self.base_url}/tools/list") as resp:
            data = await resp.json()
            return data.get("tools", [])

    async def get_tool_schema(self, name: str) -> Dict[str, Any]:
        """
        获取工具的 Schema

        Args:
            name: 工具名称

        Returns:
            工具 Schema 定义
        """
        session = await self._get_session()

        async with session.get(
            f"{self.base_url}/tools/schema",
            params={"name": name},
        ) as resp:
            return await resp.json()

    async def close(self):
        """关闭 HTTP 会话"""
        if self._session and not self._session.closed:
            await self._session.close()

    async def __aenter__(self):
        """支持 async with 语法"""
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """支持 async with 语法"""
        await self.close()


# ========== 全局桥接实例 ==========
# 在代码执行环境中,这个实例会被自动创建
_bridge: Optional[AsterBridge] = None


def _init_bridge(base_url: Optional[str] = None) -> AsterBridge:
    """初始化全局桥接实例"""
    global _bridge
    if _bridge is None:
        _bridge = AsterBridge(base_url)
    return _bridge


# ========== 工具函数 (会被动态生成) ==========
# 以下函数会在代码注入时动态生成,这里提供类型提示


async def Read(path: str) -> str:
    """
    读取文件内容

    Args:
        path: 文件路径

    Returns:
        文件内容
    """
    return await _bridge.call_tool("Read", path=path)


async def Write(path: str, content: str) -> None:
    """
    写入文件内容

    Args:
        path: 文件路径
        content: 要写入的内容
    """
    await _bridge.call_tool("Write", path=path, content=content)


async def Glob(pattern: str, path: str = ".") -> list:
    """
    文件模式匹配

    Args:
        pattern: Glob 模式
        path: 搜索路径

    Returns:
        匹配的文件路径列表
    """
    return await _bridge.call_tool("Glob", pattern=pattern, path=path)


async def Grep(pattern: str, path: str = ".", glob: Optional[str] = None) -> Any:
    """
    文件内容搜索

    Args:
        pattern: 搜索模式 (正则表达式)
        path: 搜索路径
        glob: 文件过滤模式

    Returns:
        搜索结果
    """
    params = {"pattern": pattern, "path": path}
    if glob:
        params["glob"] = glob
    return await _bridge.call_tool("Grep", **params)


async def Bash(command: str, timeout: Optional[int] = None) -> Dict[str, Any]:
    """
    执行 Bash 命令

    Args:
        command: 要执行的命令
        timeout: 超时时间(秒)

    Returns:
        执行结果 (包含 stdout, stderr, exit_code)
    """
    params = {"command": command}
    if timeout is not None:
        params["timeout"] = timeout
    return await _bridge.call_tool("Bash", **params)


# ========== 工具函数生成器 ==========
def _generate_tool_function(tool_name: str):
    """
    动态生成工具函数

    Args:
        tool_name: 工具名称

    Returns:
        async 工具函数
    """
    async def tool_func(**kwargs):
        return await _bridge.call_tool(tool_name, **kwargs)

    tool_func.__name__ = tool_name
    tool_func.__doc__ = f"Call {tool_name} tool"
    return tool_func


def inject_tools(tools: list, base_url: Optional[str] = None) -> Dict[str, Any]:
    """
    注入工具到全局命名空间

    这个函数会在代码执行前被 RuntimeManager 调用,
    将所有可用的工具注入到 globals() 中

    Args:
        tools: 工具名称列表
        base_url: 桥接服务器地址

    Returns:
        注入的工具函数字典
    """
    # 初始化桥接
    global _bridge
    _bridge = _init_bridge(base_url)

    # 生成工具函数
    injected = {}
    for tool_name in tools:
        func = _generate_tool_function(tool_name)
        injected[tool_name] = func

    return injected
