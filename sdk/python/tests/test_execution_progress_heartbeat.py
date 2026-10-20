import asyncio

import pytest

from agentfield.agent import Agent
from agentfield.client import AgentFieldClient


@pytest.mark.asyncio
async def test_live_reasoner_progress_heartbeat_and_terminal_callback(monkeypatch):
    agent=Agent(node_id="swe-planner", agentfield_server="http://control", auto_register=False)
    agent._execution_heartbeat_interval_seconds=0.05
    agent.async_config.default_execution_timeout=2.0
    calls=[]
    class Resp:
        def __init__(self, status): self.status_code=status

    async def fake_request(self,method,url,**kwargs):
        calls.append((method,url,kwargs.get("json")))
        return Resp(204 if url.endswith("/heartbeat") else 200)

    monkeypatch.setattr(AgentFieldClient,"_async_request",fake_request)
    async def reasoner():
        await asyncio.sleep(0.7)
        return {"result":"completed"}
    await agent._execute_async_with_callback(reasoner_coro=reasoner,
        execution_id="exec-progress-test",reasoner_name="architect")
    heartbeats=[c for c in calls if c[1].endswith("/heartbeat")]
    completed=[c for c in calls if c[1].endswith("/status")]
    assert heartbeats
    assert all(c[2]["node_id"]=="swe-planner" and
               c[2]["instance_id"]==agent.agent_instance_id for c in heartbeats)
    assert len(completed)==1
    assert completed[0][2]["status"]=="succeeded"
    assert completed[0][2]["result"]["result"]=="completed"
    assert len(agent._pause_clocks)==0
    snapshot=len(calls)
    await asyncio.sleep(.15)
    assert len(calls)==snapshot, "no heartbeat after terminal callback"


@pytest.mark.asyncio
async def test_expired_reasoner_is_cancelled_without_heartbeat_after_terminal(monkeypatch):
    agent=Agent(node_id="swe-planner",agentfield_server="http://control",auto_register=False)
    agent._execution_heartbeat_interval_seconds=0.05
    agent.async_config.default_execution_timeout=0.2
    seen=[]
    class Resp:
        status_code=204
    async def fake_request(self, method,url,**kwargs):
        seen.append((url,kwargs.get("json")))
        return Resp()
    monkeypatch.setattr(AgentFieldClient,"_async_request",fake_request)
    async def reasoner():
        await asyncio.sleep(2)
        return "must not succeed"
    await agent._execute_async_with_callback(reasoner_coro=reasoner,
        execution_id="exec-deadline",reasoner_name="architect")
    assert [p["status"] for u,p in seen if u.endswith("/status")] == ["failed"]
    count=len(seen)
    await asyncio.sleep(.15)
    assert len(seen)==count
