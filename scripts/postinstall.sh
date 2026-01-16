#!/bin/sh
# Post-installation script for HyperTunnel

set -e

# Create symbolic link for shorter command if it doesn't exist
if [ ! -e /usr/bin/hypertunnel ] && [ -e /usr/bin/ht ]; then
    ln -sf /usr/bin/ht /usr/bin/hypertunnel
fi

# Print installation success message
echo "HyperTunnel has been successfully installed!"
echo "Run 'ht --help' or 'hypertunnel --help' to get started."

exit 0
