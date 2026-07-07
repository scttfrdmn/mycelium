function handler(event) {
    var request = event.request;
    var uri = request.uri;

    // Root or directory path -> index.html
    if (uri === '' || uri.endsWith('/')) {
        request.uri = uri + 'index.html';
        return request;
    }

    // Extract the last path segment
    var lastSegment = uri.substring(uri.lastIndexOf('/') + 1);

    // No file extension -> clean URL, serve the .html file
    // (VitePress emits guides/python-sdk.html for /guides/python-sdk)
    if (lastSegment.indexOf('.') === -1) {
        request.uri = uri + '.html';
    }

    return request;
}
