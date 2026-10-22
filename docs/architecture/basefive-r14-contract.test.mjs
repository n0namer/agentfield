import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const root = new URL('../../', import.meta.url);
const contractPath = 'docs/architecture/basefive-r14-contracts.json';
const contract = JSON.parse(readFileSync(new URL(contractPath, root), 'utf8'));
const sourcePaths = {
  handler: 'control-plane/internal/handlers/execute.go',
  storage: 'control-plane/internal/storage/execution_idempotency.go',
  graph: 'graphify-out/graph.json',
  archmodel: '.archsteer/model.json',
  archdoc: '.archsteer/architecture.md'
};
const diagramPaths = [
  'docs/architecture/basefive-r14-C4-context.md',
  'docs/architecture/basefive-r14-container.mmd',
  'docs/architecture/basefive-r14-component.mmd'
];
const bytes = Object.fromEntries(Object.entries(sourcePaths).map(([id, path]) => [id, readFileSync(new URL(path, root))]));
const source = Object.fromEntries(Object.entries(bytes).map(([id, content]) => [id, content.toString('utf8').replaceAll('\r\n', '\n')]));
const documents = Object.fromEntries(diagramPaths.map(path => [path, readFileSync(new URL(path, root), 'utf8')]));
const hash = content => createHash('sha256').update(content).digest('hex');
const normalized = text => text.replace(/\s+/g, ' ').trim();

function structBody(text, name) {
  const start = text.indexOf(`type ${name} struct {`);
  assert.ok(start >= 0, `missing struct ${name}`);
  const end = text.indexOf('\n}', start);
  assert.ok(end > start, `unterminated struct ${name}`);
  return text.slice(start, end);
}

function fields(text, name) {
  return [...structBody(text, name).matchAll(/^\s*(\w+)\s+(.+?)\s+`([^`]+)`/gm)].map(match => {
    const json = /json:"([^"]+)"/.exec(match[3]);
    assert.ok(json, `missing JSON tag ${name}.${match[1]}`);
    const [wireName, ...options] = json[1].split(',');
    return [match[1], match[2].trim(), wireName, options.includes('omitempty'), /binding:"([^"]+)"/.exec(match[3])?.[1] ?? ''];
  });
}

function parseDiagram(text, markdown) {
  if (markdown) {
    const blocks = [...text.matchAll(/```mermaid\n([\s\S]*?)```/g)];
    assert.equal(blocks.length, 1, 'one context diagram');
    text = blocks[0][1];
  }
  const lines = text.trim().split('\n').map(line => line.trim());
  assert.equal(lines.shift(), 'flowchart LR');
  const nodes = [];
  const edges = [];
  for (const line of lines) {
    const node = /^(\w+)\["([^"\n]+)"\]$/.exec(line);
    const edge = /^(\w+) -->\|([^|\n]+)\| (\w+)$/.exec(line);
    assert.ok(node || edge, `unsupported Mermaid statement: ${line}`);
    if (node) nodes.push({ id: node[1], label: node[2] });
    else edges.push({ from: edge[1], to: edge[3], label: edge[2] });
  }
  assert.equal(new Set(nodes.map(node => node.id)).size, nodes.length, 'duplicate node');
  const ids = new Set(nodes.map(node => node.id));
  for (const edge of edges) assert.ok(ids.has(edge.from) && ids.has(edge.to), 'dangling edge');
  assert.equal(new Set(edges.map(edge => JSON.stringify(edge))).size, edges.length, 'duplicate edge');
  return { nodes, edges };
}

function validate(candidate, texts = source, diagrams = documents, rawBytes = bytes) {
  assert.equal(candidate.schema, 'agentfield.basefive-r14.architecture-contract.v1');
  assert.equal(candidate.acceptance, 'UNACCEPTED');
  assert.equal(candidate.project, 'agentfield');
  assert.deepEqual(candidate.sources.map(entry => entry.id), Object.keys(sourcePaths));
  for (const entry of candidate.sources) {
    assert.equal(entry.path, sourcePaths[entry.id], 'authorized source path');
    assert.match(entry.sha256, /^[a-f0-9]{64}$/);
    assert.equal(hash(rawBytes[entry.id]), entry.sha256, `source drift: ${entry.path}`);
    if (entry.package) assert.ok(texts[entry.id].startsWith(`package ${entry.package}\n`));
  }
  assert.equal(candidate.coverage.graphify.status, 'PARTIAL_UNVERIFIED');
  assert.equal(candidate.coverage.graphify.source_fingerprints, 'UNVERIFIED');
  assert.equal(candidate.coverage.graphify.call_edges, 'UNVERIFIED');
  assert.equal(candidate.coverage.graphify.repository_completeness, 'UNVERIFIED');
  assert.equal(candidate.coverage.archsteer.status, 'PARTIAL_UNVERIFIED');
  assert.equal(candidate.coverage.archsteer.comparison, 'INVENTORY_ONLY');
  assert.equal(candidate.coverage.archsteer.source_fingerprints, 'UNVERIFIED');
  assert.equal(candidate.coverage.archsteer.go_semantics, 'UNVERIFIED');
  assert.equal(candidate.coverage.archsteer.model_schema_version, JSON.parse(texts.archmodel).schema_version);
  assert.equal(candidate.coverage.archsteer.model_commit_sha, JSON.parse(texts.archmodel).commit_sha);
  assert.equal(candidate.coverage.go_blind_spots.length, 6);
  assert.deepEqual(candidate.coverage.inventory, {
    graph_nodes: coverage.nodes, graph_go_nodes: coverage.go_nodes, graph_go_source_paths: coverage.go_source_paths,
    graph_targets: coverage.targets.map(({ path, nodes, package_source_paths }) => ({ path, nodes, package_source_paths })),
    arch_components: archCoverage.components, arch_go_components: archCoverage.go_components,
    arch_languages: archCoverage.languages, arch_targets: archCoverage.targets,
    arch_document_mentions: archCoverage.document_mentions, arch_component_fingerprint_fields: archCoverage.fingerprint_fields
  }, 'coverage inventory drift');
  assert.equal(candidate.api.wire_version, 'UNVERSIONED_IN_AUTHORIZED_SOURCE');
  assert.equal(candidate.api.router, 'UNVERIFIED');
  assert.deepEqual(candidate.api.handlers.map(entry => entry.name), ['ExecuteHandler', 'ExecuteAsyncHandler', 'GetExecutionStatusHandler', 'BatchExecutionStatusHandler', 'ExecuteHandlerWithARD', 'UpdateExecutionStatusHandler']);
  for (const entry of candidate.api.handlers) {
    const signature = `func ${entry.name}(${entry.signature}) ${entry.returns} {`;
    assert.ok(texts.handler.includes(signature), `handler signature ${entry.name}`);
    const body = texts.handler.slice(texts.handler.indexOf(signature)).split('\n}')[0];
    assert.ok(body.includes(`return controller.${entry.delegate}`), `handler delegate ${entry.name}`);
  }
  assert.deepEqual(candidate.api.dtos.map(entry => entry.name), ['ExecuteRequest', 'WebhookRequest', 'ExecuteResponse', 'AsyncExecuteResponse', 'ExecutionStatusResponse', 'BatchStatusRequest', 'executionStatusUpdateRequest']);
  for (const dto of candidate.api.dtos) assert.deepEqual(fields(texts.handler, dto.name), dto.fields, `DTO drift ${dto.name}`);
  assert.deepEqual(candidate.api.aliases, ['type BatchStatusResponse map[string]ExecutionStatusResponse']);
  for (const alias of candidate.api.aliases) assert.ok(texts.handler.includes(alias));
  const interfaceStart = texts.storage.indexOf('type AsyncExecutionIdempotencyStore interface {');
  assert.ok(interfaceStart >= 0, 'durable interface');
  const methods = texts.storage.slice(interfaceStart).split('\n}')[0].split('\n').slice(1).map(normalized).filter(Boolean);
  assert.deepEqual(methods, candidate.api.storage_interface);
  const expectedHttp = { invalid_key: 400, unauthenticated: 401, unavailable_store: 503, conflict: 409, replay: 202, queued: 202 };
  assert.deepEqual(candidate.protocol.http, expectedHttp);
  assert.equal(candidate.protocol.max_key_bytes, Number(/maxAsyncIdempotencyKeyLength\s*=\s*(\d+)/.exec(texts.handler)?.[1]));
  assert.deepEqual(candidate.protocol.ascii_range, [0x21, 0x7e]);
  assert.equal(candidate.protocol.header, 'Idempotency-Key');
  assert.equal(candidate.protocol.replay_header, 'X-AgentField-Idempotency-Replay');
  assert.equal(candidate.protocol.digest, 'SHA-256 of raw request body bytes; includes whitespace and run_metadata');
  assert.deepEqual(candidate.protocol.unique_key, ['scope_key', 'request_key']);
  assert.deepEqual(candidate.protocol.conflict_on, ['target', 'body_digest']);
  assert.equal(candidate.protocol.scope, 'trimmed verified caller DID + NUL + trimmed target DID');
  assert.equal(candidate.protocol.replay_type, 'empty string (Type is not populated but has no omitempty)');
  assert.equal(candidate.protocol.replay_enqueued_at, 'CreatedAt, not independently persisted queue time');
  assert.equal(candidate.protocol.queue, 'process-local channel; durable binding does not prove restart dispatch');
  assert.equal(candidate.protocol.anchors.length, 13);
  for (const anchor of candidate.protocol.anchors) {
    assert.ok(['handler', 'storage'].includes(anchor.source));
    assert.ok(normalized(texts[anchor.source]).includes(normalized(anchor.text)), `protocol anchor ${anchor.text}`);
  }
  const reader = texts.handler.slice(texts.handler.indexOf('func readAsyncIdempotencyRequest(')).split('\n}')[0];
  assert.ok(reader.includes('io.ReadAll(ctx.Request.Body)') && reader.includes('sha256.Sum256(body)'));
  const replay = texts.handler.slice(texts.handler.indexOf('func writeAsyncIdempotencyResponse(')).split('\n}')[0];
  assert.ok(replay.includes('ctx.JSON(http.StatusAccepted, response)'));
  assert.ok(!/^\s*Type:/m.test(replay), 'replay Type remains zero value');
  assert.ok(replay.includes('EnqueuedAt:        createdAt'));
  assert.deepEqual(candidate.diagrams.map(entry => entry.level), ['context', 'container', 'component']);
  assert.deepEqual(candidate.diagrams.map(entry => entry.path), diagramPaths);
  const requiredNodes = { context: ['Caller', 'Agentfield', 'Agent'], container: ['Client', 'ControlPlane', 'Database', 'Agent'], component: ['AsyncHandler', 'KeyReader', 'Adapter', 'DurableStore', 'SQL', 'Pool'] };
  const requiredEdgeCounts = { context: 2, container: 3, component: 6 };
  for (const diagram of candidate.diagrams) {
    assert.deepEqual(diagram.nodes.map(node => node.id), requiredNodes[diagram.level], 'required architecture spine');
    assert.equal(diagram.edges.length, requiredEdgeCounts[diagram.level], 'required relationships');
    const parsed = parseDiagram(diagrams[diagram.path], diagram.level === 'context');
    assert.deepEqual(parsed.nodes, diagram.nodes.map(({ id, label }) => ({ id, label })), 'diagram node drift');
    assert.deepEqual(parsed.edges, diagram.edges.map(({ from, to, label }) => ({ from, to, label })), 'diagram edge drift');
    for (const element of [...diagram.nodes, ...diagram.edges]) {
      assert.ok(['handler', 'storage'].includes(element.source), 'source-backed diagram reference');
      assert.ok(element.anchor.length > 0 && texts[element.source].includes(element.anchor), `diagram anchor ${element.anchor}`);
    }
  }
}

const graph = JSON.parse(source.graph);
assert.ok(Array.isArray(graph.nodes), 'Graphify nodes inventory');
const pathOf = node => typeof node.source_file === 'string' ? node.source_file.replaceAll('\\', '/') : '';
const goNodes = graph.nodes.filter(node => pathOf(node).endsWith('.go'));
const goPaths = new Set(goNodes.map(pathOf));
const coverage = {
  graph_sha256: hash(bytes.graph),
  nodes: graph.nodes.length,
  go_nodes: goNodes.length,
  go_source_paths: goPaths.size,
  targets: Object.entries(sourcePaths).filter(([id]) => ['handler', 'storage'].includes(id)).map(([id, path]) => {
    const nodes = graph.nodes.filter(node => pathOf(node) === path);
    const directory = path.slice(0, path.lastIndexOf('/') + 1);
    return { id, path, nodes: nodes.length, package_source_paths: [...goPaths].filter(value => value.startsWith(directory)).length, languages: [...new Set(nodes.map(node => node.metadata?.language ?? 'UNKNOWN'))], kinds: [...new Set(nodes.map(node => node.metadata?.kind ?? 'UNKNOWN'))], labels_sample: nodes.slice(0, 6).map(node => node.label) };
  })
};
console.log(`GRAPHIFY_COVERAGE ${JSON.stringify(coverage)}`);
const model = JSON.parse(source.archmodel);
assert.ok(model.components && typeof model.components === 'object');
const components = Object.entries(model.components);
const archCoverage = {
  model_sha256: hash(bytes.archmodel),
  architecture_sha256: hash(bytes.archdoc),
  schema_version: model.schema_version,
  commit_sha: model.commit_sha,
  components: components.length,
  languages: Object.fromEntries([...new Set(components.map(([, component]) => component.language ?? 'UNKNOWN'))].map(language => [language, components.filter(([, component]) => (component.language ?? 'UNKNOWN') === language).length])),
  go_components: components.filter(([path, component]) => path.endsWith('.go') || component.language === 'go').length,
  targets: coverage.targets.map(target => ({ path: target.path, component_present: Object.hasOwn(model.components, target.path), package_components: components.filter(([path]) => path.startsWith(target.path.slice(0, target.path.lastIndexOf('/') + 1))).length })),
  document_mentions: coverage.targets.map(target => ({ path: target.path, present: source.archdoc.includes(target.path) })),
  fingerprint_fields: [...new Set(components.flatMap(([, component]) => Object.keys(component).filter(key => /sha|hash|fingerprint/i.test(key))))]
};
console.log(`ARCHSTEER_COVERAGE ${JSON.stringify(archCoverage)}`);

test('source-bound API/DTO/protocol and all three Mermaid inventories', () => validate(contract));
test('Go graph inventory is bounded and does not masquerade as semantic coverage', () => {
  assert.ok(coverage.go_nodes > 0, 'graph contains Go paths');
  assert.equal(coverage.targets.length, 2);
  for (const target of coverage.targets) assert.ok(target.nodes >= 0 && target.package_source_paths >= 0);
  assert.equal(contract.coverage.graphify.source_fingerprints, 'UNVERIFIED');
});

const mutations = [
  ['stale source fingerprint', candidate => { candidate.sources[0].sha256 = '0'.repeat(64); }],
  ['unauthorized source path', candidate => { candidate.sources[0].path = '../execute.go'; }],
  ['self acceptance', candidate => { candidate.acceptance = 'ACCEPTED_GREEN'; }],
  ['false Graphify completeness', candidate => { candidate.coverage.graphify.repository_completeness = 'VERIFIED'; }],
  ['false ArchSteer comparison', candidate => { candidate.coverage.archsteer.comparison = 'VERIFIED'; }],
  ['invented API version', candidate => { candidate.api.wire_version = 'v1'; }],
  ['false indexed storage presence', candidate => { candidate.coverage.inventory.graph_targets[1].nodes = 1; }],
  ['false ArchSteer Go coverage', candidate => { candidate.coverage.inventory.arch_go_components = 1; }],
  ['wrong DID scope', candidate => { candidate.protocol.scope = 'request key only'; }],
  ['invented durable queue', candidate => { candidate.protocol.queue = 'restart safe'; }],
  ['wrong handler signature', candidate => { candidate.api.handlers[1].signature = 'store any'; }],
  ['wrong delegate', candidate => { candidate.api.handlers[1].delegate = 'handleSync'; }],
  ['missing DTO', candidate => { candidate.api.dtos.pop(); }],
  ['missing DTO field', candidate => { candidate.api.dtos[3].fields.pop(); }],
  ['wrong JSON name', candidate => { candidate.api.dtos[3].fields[0][2] = 'id'; }],
  ['wrong Go field type', candidate => { candidate.api.dtos[3].fields[0][1] = 'int'; }],
  ['wrong omitempty', candidate => { candidate.api.dtos[3].fields[8][3] = true; }],
  ['wrong required binding', candidate => { candidate.api.dtos[5].fields[0][4] = ''; }],
  ['missing durable method', candidate => { candidate.api.storage_interface.pop(); }],
  ['wrong byte limit', candidate => { candidate.protocol.max_key_bytes = 256; }],
  ['wrong conflict status', candidate => { candidate.protocol.http.conflict = 200; }],
  ['canonical-JSON digest fiction', candidate => { candidate.protocol.digest = 'canonical JSON excluding run_metadata'; }],
  ['wrong SQL unique key', candidate => { candidate.protocol.unique_key = ['target', 'request_key']; }],
  ['invented diagram source anchor', candidate => { candidate.diagrams[2].edges[0].anchor = 'unknownCall()'; }]
];
for (const [name, mutate] of mutations) {
  test(`reject ${name}`, () => {
    const candidate = structuredClone(contract);
    mutate(candidate);
    assert.throws(() => validate(candidate));
  });
}

for (const [name, mutate] of [
  ['extra undocumented node', text => `${text}  Ghost["invented"]\n`],
  ['dangling edge', text => `${text}  Pool -->|fabricated| Ghost\n`],
  ['duplicate node', text => `${text}  Pool["asyncWorkerPool / process-local queue"]\n`],
  ['invented edge between real nodes', text => `${text}  SQL -->|restart dispatch not proved| Pool\n`],
  ['wrong node label', text => text.replace('process-local queue', 'durable restart queue')],
  ['unsupported Mermaid syntax', text => `${text}  click Pool "https://example.invalid"\n`]
]) {
  test(`reject Mermaid ${name}`, () => {
    const changed = { ...documents, [diagramPaths[2]]: mutate(documents[diagramPaths[2]]) };
    assert.throws(() => validate(contract, source, changed));
  });
}

test('raw-body identity distinguishes whitespace and metadata changes', () => {
  assert.notEqual(hash('{"input":{}}'), hash('{ "input": {} }'));
  assert.notEqual(hash('{"input":{},"run_metadata":{"name":"a"}}'), hash('{"input":{},"run_metadata":{"name":"b"}}'));
});
test('reject independently changed DTO source even if fingerprint is rebound', () => {
  const candidate = structuredClone(contract);
  const changedText = source.handler.replace('json:"execution_id"', 'json:"renamed_execution_id"');
  const changedBytes = { ...bytes, handler: Buffer.from(changedText) };
  candidate.sources[0].sha256 = hash(changedBytes.handler);
  assert.throws(() => validate(candidate, { ...source, handler: changedText }, documents, changedBytes), /DTO drift/);
});
test('diagram verifier rejects missing context Mermaid fence', () => {
  assert.throws(() => parseDiagram('# no diagram', true));
});
test('scoped artifact integrity and immutable readback hashes', () => {
  const paths = [...diagramPaths, contractPath, 'docs/architecture/basefive-r14-drift-analysis.md', 'docs/architecture/basefive-r14-contract.test.mjs'];
  const artifacts = paths.map(path => {
    const content = readFileSync(new URL(path, root));
    assert.ok(content.length > 0, `nonempty artifact ${path}`);
    assert.ok(content.toString('utf8').split('\n').length < 500, `line budget ${path}`);
    return { path, sha256: hash(content), bytes: content.length };
  });
  console.log(`ARTIFACT_RECEIPT ${JSON.stringify({ recorded_at: new Date().toISOString(), artifacts })}`);
});

test('runtime and release gates remain explicitly unverified', () => {
  const drift = readFileSync(new URL('docs/architecture/basefive-r14-drift-analysis.md', root), 'utf8');
  assert.ok(drift.includes('UNACCEPTED') && drift.includes('Go runtime') && drift.includes('UNVERIFIED'));
});
