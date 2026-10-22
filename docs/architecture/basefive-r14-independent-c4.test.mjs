import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import './basefive-r14-contract.test.mjs';

const root = new URL('../../', import.meta.url);
const load = path => readFileSync(new URL(path, root));
const hash = value => createHash('sha256').update(value).digest('hex');
const text = value => value.toString('utf8').replaceAll('\r\n', '\n');
const sourceBindings = [
  ['handler', 'control-plane/internal/handlers/execute.go', 'be09150579e29e16ce9d96775ef1b161898d829570b9bdd5885c9db5327a25c1'],
  ['storage', 'control-plane/internal/storage/execution_idempotency.go', '7ff044cdce4e1ec772e3e79a6d4e074e9fece17c2a6857bdf0f4552a1db9cfe2'],
  ['graph', 'graphify-out/graph.json', '1f00b4b243e4f4ec7e9fb35a8373c2054f95615fe4a454ab8e02addee1cc8f5d'],
  ['archmodel', '.archsteer/model.json', '0350f5afc09c996266c2745fb27e2fd2d29e3245e15f1568f8f6681206f17f04'],
  ['archdoc', '.archsteer/architecture.md', 'af90dfa9b7d59211a5d90d0fe057d569d841ef34d7c1c177f446e957584d4c5a']
];
const bytes = Object.fromEntries(sourceBindings.map(([id, path]) => [id, load(path)]));
const sources = Object.fromEntries(Object.entries(bytes).map(([id, value]) => [id, text(value)]));
const contractPath = 'docs/architecture/basefive-r14-contracts.json';
const contract = JSON.parse(text(load(contractPath)));
const diagramPaths = [
  'docs/architecture/basefive-r14-C4-context.md',
  'docs/architecture/basefive-r14-container.mmd',
  'docs/architecture/basefive-r14-component.mmd'
];
const diagrams = Object.fromEntries(diagramPaths.map(path => [path, text(load(path))]));
const dtoNames = ['ExecuteRequest', 'WebhookRequest', 'ExecuteResponse', 'AsyncExecuteResponse', 'ExecutionStatusResponse', 'BatchStatusRequest', 'executionStatusUpdateRequest'];
const expectedEdges = [
  [['Caller', 'Agentfield', 'Execution request / status'], ['Agentfield', 'Agent', 'Agent call [implementation unverified]']],
  [['Client', 'ControlPlane', 'Execute DTOs / status'], ['ControlPlane', 'Database', 'Optional durable idempotency + execution transaction'], ['ControlPlane', 'Agent', 'Agent call [implementation unverified]']],
  [['AsyncHandler', 'KeyReader', 'Validate key / caller / raw-body digest'], ['AsyncHandler', 'DurableStore', 'Lookup before admission'], ['AsyncHandler', 'Adapter', 'Prepare with optional durable adapter'], ['Adapter', 'DurableStore', 'CreateAsyncExecutionIdempotent'], ['DurableStore', 'SQL', 'Atomic binding + execution insert'], ['AsyncHandler', 'Pool', 'Reserve / submit job']]
];
const componentOwners = [
  ['handler', 'func (c *executionController) handleAsync(', 'readAsyncIdempotencyRequest(ctx)'],
  ['handler', 'func (c *executionController) handleAsync(', 'idempotencyStore.LookupAsyncExecutionIdempotency('],
  ['handler', 'func (c *executionController) handleAsync(', 'prepareCopy.store = &asyncIdempotentExecutionStore{'],
  ['handler', 'func (s *asyncIdempotentExecutionStore) CreateExecutionRecord(', 's.store.CreateAsyncExecutionIdempotent('],
  ['storage', 'func (ls *LocalStorage) CreateAsyncExecutionIdempotent(', 'insertExecutionRecordTx(ctx, tx, execution)'],
  ['handler', 'func (c *executionController) handleAsync(', 'pool.submitReserved(job)']
];

function declaration(source, prefix) {
  const start = source.indexOf(prefix);
  assert.ok(start >= 0, `missing declaration: ${prefix}`);
  const end = source.indexOf('\n}', start);
  assert.ok(end > start, `missing declaration end: ${prefix}`);
  return source.slice(start, end + 2);
}

function dtoFields(source, name) {
  return declaration(source, `type ${name} struct {`).split('\n').slice(1, -1)
    .map(line => line.trim()).filter(line => line && !line.startsWith('//')).map(line => {
      const match = /^(\w+)\s+(.+?)\s+`([^`]+)`$/.exec(line);
      assert.ok(match, `unsupported DTO field: ${name}.${line}`);
      const jsonTag = /(?:^|\s)json:"([^"]+)"/.exec(match[3]);
      assert.ok(jsonTag, `missing JSON tag: ${name}.${match[1]}`);
      const [wire, ...options] = jsonTag[1].split(',');
      return [match[1], match[2].trim(), wire, options.includes('omitempty'), /binding:"([^"]+)"/.exec(match[3])?.[1] ?? ''];
    });
}

function validateBindings(candidate, raw = bytes) {
  assert.deepEqual(candidate.sources.map(({ id, path }) => [id, path]), sourceBindings.map(([id, path]) => [id, path]));
  for (const [id, path, expected] of sourceBindings) {
    assert.equal(hash(raw[id]), expected, `independent snapshot drift: ${path}`);
    assert.equal(candidate.sources.find(entry => entry.id === id).sha256, expected, `contract SHA drift: ${path}`);
  }
}

function validateDTOs(candidate, source = sources.handler) {
  assert.deepEqual(candidate.api.dtos.map(dto => dto.name), dtoNames, 'complete local DTO set');
  for (const dto of candidate.api.dtos) assert.deepEqual(dto.fields, dtoFields(source, dto.name), `independent DTO drift: ${dto.name}`);
  assert.equal(candidate.api.wire_version, 'UNVERSIONED_IN_AUTHORIZED_SOURCE');
  assert.equal(candidate.api.router, 'UNVERIFIED');
}

function validateEdges(candidate, documents = diagrams, source = sources) {
  assert.deepEqual(candidate.diagrams.map(diagram => diagram.path), diagramPaths);
  for (const [index, diagram] of candidate.diagrams.entries()) {
    assert.deepEqual(diagram.edges.map(({ from, to, label }) => [from, to, label]), expectedEdges[index], 'independent edge topology drift');
    const document = documents[diagram.path];
    const diagramText = index === 0 ? /```mermaid\n([\s\S]*?)```/.exec(document)?.[1] : document;
    assert.ok(diagramText, 'diagram body');
    const parsedEdges = diagramText.split('\n').map(line => line.trim()).filter(line => line.includes('-->')).map(line => {
      const match = /^(\w+) -->\|([^|]+)\| (\w+)$/.exec(line);
      assert.ok(match, `unsupported edge: ${line}`);
      return [match[1], match[3], match[2]];
    });
    assert.deepEqual(parsedEdges, expectedEdges[index], 'independent rendered edge drift');
  }
  for (const [index, [id, owner, anchor]] of componentOwners.entries()) {
    const edge = candidate.diagrams[2].edges[index];
    assert.equal(edge.source, id, 'component edge source');
    assert.equal(edge.anchor, anchor, 'component edge anchor');
    assert.ok(declaration(source[id], owner).includes(anchor), `edge not owned by ${owner}`);
  }
  const asyncBody = declaration(source.handler, 'func (c *executionController) handleAsync(');
  assert.ok(asyncBody.indexOf('idempotencyStore.LookupAsyncExecutionIdempotency(') < asyncBody.indexOf('pool.reserveForAdmission()'), 'lookup precedes reservation');
  const createBody = declaration(source.storage, 'func (ls *LocalStorage) CreateAsyncExecutionIdempotent(');
  assert.ok(createBody.includes('ON CONFLICT (scope_key, request_key) DO NOTHING'));
  assert.ok(createBody.indexOf('insertExecutionRecordTx(ctx, tx, execution)') < createBody.lastIndexOf('tx.Commit()'), 'execution insert precedes final commit');
}

test('independent exact source SHA bindings', () => validateBindings(contract));
test('independent seven local Go DTO declarations', () => validateDTOs(contract));
test('independent source-owned relationships across three C4 levels', () => validateEdges(contract));

test('negative DTO wire-name drift is rejected against unchanged Go source', () => {
  const changed = structuredClone(contract);
  changed.api.dtos.find(dto => dto.name === 'AsyncExecuteResponse').fields[0][2] = 'id';
  assert.throws(() => validateDTOs(changed), /independent DTO drift: AsyncExecuteResponse/);
});

test('negative paired diagram and contract edge drift is rejected even with a real source anchor', () => {
  const changed = structuredClone(contract);
  changed.diagrams[2].edges[4].to = 'Pool';
  const changedDocuments = { ...diagrams, [diagramPaths[2]]: diagrams[diagramPaths[2]].replace('DurableStore -->|Atomic binding + execution insert| SQL', 'DurableStore -->|Atomic binding + execution insert| Pool') };
  assert.ok(sources.storage.includes(changed.diagrams[2].edges[4].anchor));
  assert.throws(() => validateEdges(changed, changedDocuments), /independent edge topology drift/);
});

test('negative source drift cannot be hidden by rebinding its contract fingerprint', () => {
  const changed = structuredClone(contract);
  const raw = { ...bytes, handler: Buffer.from(sources.handler.replace('json:"execution_id"', 'json:"renamed_execution_id"')) };
  changed.sources[0].sha256 = hash(raw.handler);
  assert.throws(() => validateBindings(changed, raw), /independent snapshot drift/);
});

test('negative source call drift is rejected even when the original anchor exists elsewhere', () => {
  const changed = { ...sources, handler: sources.handler.replace('pool.submitReserved(job)', 'pool.notSubmitted(job)') };
  changed.handler += '\n// pool.submitReserved(job)\n';
  assert.ok(changed.handler.includes('pool.submitReserved(job)'));
  assert.throws(() => validateEdges(contract, diagrams, changed), /edge not owned/);
});

test('independent Graphify and ArchSteer Go/Python inventory and limits', () => {
  const graph = JSON.parse(sources.graph);
  const model = JSON.parse(sources.archmodel);
  const pathOf = node => typeof node.source_file === 'string' ? node.source_file.replaceAll('\\', '/') : '';
  const goNodes = graph.nodes.filter(node => pathOf(node).endsWith('.go'));
  const goPaths = [...new Set(goNodes.map(pathOf))];
  const components = Object.entries(model.components);
  const languages = Object.fromEntries([...new Set(components.map(([, component]) => component.language ?? 'UNKNOWN'))].map(language => [language, components.filter(([, component]) => (component.language ?? 'UNKNOWN') === language).length]));
  const targets = sourceBindings.slice(0, 2).map(([, path]) => {
    const directory = path.slice(0, path.lastIndexOf('/') + 1);
    return { path, nodes: graph.nodes.filter(node => pathOf(node) === path).length, package_source_paths: goPaths.filter(value => value.startsWith(directory)).length };
  });
  const inventory = {
    graph_nodes: graph.nodes.length, graph_go_nodes: goNodes.length, graph_go_source_paths: goPaths.length, graph_targets: targets,
    arch_components: components.length, arch_go_components: components.filter(([path, component]) => path.endsWith('.go') || component.language === 'go').length,
    arch_languages: languages,
    arch_targets: targets.map(({ path }) => ({ path, component_present: Object.hasOwn(model.components, path), package_components: components.filter(([value]) => value.startsWith(path.slice(0, path.lastIndexOf('/') + 1))).length })),
    arch_document_mentions: targets.map(({ path }) => ({ path, present: sources.archdoc.includes(path) })),
    arch_component_fingerprint_fields: [...new Set(components.flatMap(([, component]) => Object.keys(component).filter(key => /sha|hash|fingerprint/i.test(key))))]
  };
  assert.deepEqual(inventory, contract.coverage.inventory, 'independent coverage drift');
  assert.deepEqual(targets.map(target => target.nodes), [43, 0]);
  assert.equal(inventory.arch_go_components, 0);
  assert.equal(languages.python, 447);
  assert.equal(model.schema_version, '1.0.0');
  assert.equal(model.commit_sha, '44167dc1');
  assert.equal(contract.coverage.graphify.call_edges, 'UNVERIFIED');
  assert.equal(contract.coverage.archsteer.go_semantics, 'UNVERIFIED');
  console.log(`INDEPENDENT_COVERAGE ${JSON.stringify(inventory)}`);
});

test('independent acceptance and runtime limits remain fail closed', () => {
  assert.equal(contract.acceptance, 'UNACCEPTED');
  const frontier = text(load('docs/acceptance/c4-releasegate-reducer-frontier-20261013.md'));
  const drift = text(load('docs/architecture/basefive-r14-drift-analysis.md'));
  assert.ok(frontier.includes('BLOCKED') && frontier.includes('UNACCEPTED'));
  assert.ok(drift.includes('Go runtime') && drift.includes('UNVERIFIED'));
  assert.match(contract.protocol.queue, /process-local.*does not prove restart/);
});

test('independent immutable artifact receipt', () => {
  const paths = [...sourceBindings.map(([, path]) => path), ...diagramPaths, contractPath, 'docs/architecture/basefive-r14-drift-analysis.md', 'docs/architecture/basefive-r14-contract.test.mjs', 'docs/architecture/basefive-r14-independent-c4.test.mjs', 'docs/architecture/basefive-r14-independent-c4-review.md'];
  const artifacts = paths.map(path => {
    const content = load(path);
    assert.ok(content.length > 0, `nonempty ${path}`);
    return { path, sha256: hash(content), bytes: content.length };
  });
  console.log(`INDEPENDENT_QA_RECEIPT ${JSON.stringify({ schema: 'agentfield.basefive-r14.independent-c4-qa.v1', acceptance: 'UNACCEPTED', recorded_at: new Date().toISOString(), artifacts })}`);
});
