# Replace Plugin

"This plugin adds the /s command to find and replace a string in the user's last post

To learn more about plugins, see [our plugin documentation](https://developers.mattermost.com/extend/plugins/).

## Getting Started
Clone the repository to a directory outside of `$GOPATH` matching your plugin name:
```
git clone --depth 1 https://github.com/evan1024/mattermost-plugin-replace com.example.my-plugin
```

Build your plugin:
```
make
```

This will produce a single plugin file (with support for multiple architectures) for upload to your Mattermost server:

```
dist/com.example.my-plugin.tar.gz
```

There is a build target to automate deploying and enabling the plugin to your server, but it requires configuration and [http](https://httpie.org/) to be installed:
```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make deploy
```

Alternatively, if you are running your `mattermost-server` out of a sibling directory by the same name, use the `deploy` target alone to  unpack the files into the right directory. You will need to restart your server and manually enable your plugin.

In production, deploy and upload your plugin via the [System Console](https://about.mattermost.com/default-plugin-uploads).