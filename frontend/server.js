const express = require('express');
const path = require('path');
const chokidar = require('chokidar');
const WebSocket = require('ws');
const fs = require('fs');

const app = express();
const PORT = process.env.PORT || 3001;

// Serve static files from current directory
app.use(express.static(__dirname));

// Create HTTP server
const server = require('http').createServer(app);

// WebSocket server for live reload
const wss = new WebSocket.Server({ server });

// Store connected clients
const clients = new Set();

// Handle WebSocket connections
wss.on('connection', (ws) => {
    console.log('🔌 Client connected for live reload');
    clients.add(ws);
    
    ws.on('close', () => {
        console.log('🔌 Client disconnected');
        clients.delete(ws);
    });
});

// Function to notify all clients to reload
function notifyReload(filename) {
    const message = JSON.stringify({ type: 'reload', file: filename });
    clients.forEach(client => {
        if (client.readyState === WebSocket.OPEN) {
            client.send(message);
        }
    });
    console.log(`🔄 Notified ${clients.size} clients to reload (${filename} changed)`);
}

// Watch for file changes
const watcher = chokidar.watch([
    '*.html',
    '*.js',
    '*.css',
    'components/*.js'
], {
    ignored: /node_modules/,
    persistent: true,
    ignoreInitial: true
});

watcher.on('change', (path) => {
    console.log(`📝 File changed: ${path}`);
    notifyReload(path);
});

watcher.on('add', (path) => {
    console.log(`➕ File added: ${path}`);
    notifyReload(path);
});

watcher.on('unlink', (path) => {
    console.log(`➖ File removed: ${path}`);
    notifyReload(path);
});

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index-modular.html'));
});

// Inject live reload script into HTML files
app.get('*.html', (req, res) => {
    const filePath = path.join(__dirname, req.path);
    
    if (fs.existsSync(filePath)) {
        let html = fs.readFileSync(filePath, 'utf8');
        
        // Inject live reload script before closing body tag
        const liveReloadScript = `
<script>
(function() {
    const ws = new WebSocket('ws://localhost:${PORT}');
    
    ws.onopen = function() {
        console.log('🔄 Live reload connected');
    };
    
    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        if (data.type === 'reload') {
            console.log('🔄 Reloading page due to file change:', data.file);
            window.location.reload();
        }
    };
    
    ws.onclose = function() {
        console.log('🔄 Live reload disconnected');
        // Try to reconnect after 1 second
        setTimeout(() => {
            window.location.reload();
        }, 1000);
    };
    
    ws.onerror = function(error) {
        console.log('🔄 Live reload error:', error);
    };
})();
</script>`;
        
        // Insert before closing body tag, or at the end if no body tag
        if (html.includes('</body>')) {
            html = html.replace('</body>', liveReloadScript + '\n</body>');
        } else {
            html += liveReloadScript;
        }
        
        res.send(html);
    } else {
        res.status(404).send('File not found');
    }
});

// API endpoint for development info
app.get('/api/dev-info', (req, res) => {
    res.json({
        message: 'QuizUS Frontend Development Server',
        port: PORT,
        connectedClients: clients.size,
        watchedFiles: [
            '*.html',
            '*.js', 
            '*.css',
            'components/*.js'
        ]
    });
});

// Start the server
server.listen(PORT, () => {
    console.log('🚀 QuizUS Frontend Development Server started!');
    console.log(`📍 Server running at: http://localhost:${PORT}`);
    console.log(`📂 Serving files from: ${__dirname}`);
    console.log('🔄 Live reload enabled - your browser will refresh when files change');
    console.log('📁 Watching for changes in:');
    console.log('   - *.html files');
    console.log('   - *.js files');
    console.log('   - *.css files');
    console.log('   - components/*.js files');
    console.log('');
    console.log('💡 To stop the server, press Ctrl+C');
    console.log('');
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('🛑 Server shutting down...');
    watcher.close();
    server.close(() => {
        console.log('✅ Server closed');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('\n🛑 Server shutting down...');
    watcher.close();
    server.close(() => {
        console.log('✅ Server closed');
        process.exit(0);
    });
});