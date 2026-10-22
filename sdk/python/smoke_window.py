import asyncio, os, sys
from agentfield.harness._cli import run_cli
async def one(i):
    out,err,rc=await run_cli(["cmd.exe","/c","echo",f"AF_NO_WINDOW_{i}"],timeout=20)
    assert rc==0 and f"AF_NO_WINDOW_{i}" in out,(i,rc,out,err)
    return rc
async def main():
    print("SMOKE_10",await asyncio.gather(*(one(i) for i in range(10))))
asyncio.run(main())
