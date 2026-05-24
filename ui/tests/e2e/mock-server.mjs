import http from 'node:http';

const view = {
  groups: [
    { id: 'media', name: 'Media', order: 0 },
    { id: 'infra', name: 'Infrastructure', order: 1 },
  ],
  tiles: [
    {
      id: 'media/jellyfin/jellyfin.example.com',
      source: 'httproute',
      name: 'Jellyfin',
      url: 'https://jellyfin.example.com',
      icon: 'jellyfin',
      group: 'media',
      order: 0,
      hidden: false,
      status: { state: 'up', statusCode: 200, latencyMs: 12, checkedAt: '' },
      k8s: { namespace: 'media', httpRouteName: 'jellyfin', gatewayRefs: [{ namespace: 'gw', name: 'ext' }] },
    },
    {
      id: 'infra/grafana/grafana.example.com',
      source: 'httproute',
      name: 'Grafana',
      url: 'https://grafana.example.com',
      icon: 'grafana',
      group: 'infra',
      order: 0,
      hidden: false,
      status: { state: 'down', statusCode: 0, latencyMs: 0, checkedAt: '', error: 'unreachable' },
    },
  ],
};

http.createServer((req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*');
  if (req.url === '/api/tiles' && req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(view));
    return;
  }
  if (req.url === '/api/events' && req.method === 'GET') {
    res.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' });
    res.write(': hello\n\n');
    return; // keep open
  }
  if (req.url?.startsWith('/icons/')) {
    // 1×1 transparent PNG so <img> doesn't 404.
    const png = Buffer.from(
      '89504e470d0a1a0a0000000d49484452000000010000000108060000001f15c4890000000d49444154789c63000100000005000146da77c30000000049454e44ae426082',
      'hex',
    );
    res.writeHead(200, { 'Content-Type': 'image/png' });
    res.end(png);
    return;
  }
  if (req.method === 'PATCH' || req.method === 'POST' || req.method === 'DELETE' || req.method === 'PUT') {
    res.writeHead(200);
    res.end();
    return;
  }
  res.writeHead(404);
  res.end();
}).listen(8080, () => console.log('mock backend on :8080'));
