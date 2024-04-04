const path = require('path');

module.exports = {
    mode: 'production',
    devtool: 'source-map',
    entry: {
        images: './images_test.js',
        randomized: './randomized_test.js'
    },
    output: {
        path: path.resolve(__dirname, 'dist'),
        libraryTarget: 'commonjs',
        filename: '[name].bundle.js',
    },
    module: {
        rules: [{test: /\.js$/, use: 'babel-loader'}],
    },
    target: 'web',
    externals: /k6(\/.*)?/,
};
