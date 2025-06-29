#!/bin/bash

# TinyOS Environment Setup Script - Linux Version

echo -e "\033[32mTinyOS Security Setup\033[0m"
echo -e "\033[32m=====================\033[0m"

# Generate secure JWT secret
echo ""
echo -e "\033[33mGenerating secure JWT secret...\033[0m"
JWT_SECRET=$(openssl rand -base64 32)

# Set JWT secret in user's profile
echo ""
echo -e "\033[33mSetting environment variables...\033[0m"

# Add to .bashrc
if ! grep -q "export JWT_SECRET_KEY=" ~/.bashrc; then
    echo "export JWT_SECRET_KEY=\"$JWT_SECRET\"" >> ~/.bashrc
    echo -e "\033[32m✓ JWT_SECRET_KEY added to ~/.bashrc\033[0m"
else
    # Update existing entry
    sed -i "s/export JWT_SECRET_KEY=.*/export JWT_SECRET_KEY=\"$JWT_SECRET\"/" ~/.bashrc
    echo -e "\033[32m✓ JWT_SECRET_KEY updated in ~/.bashrc\033[0m"
fi

# Also add to current session
export JWT_SECRET_KEY="$JWT_SECRET"

echo ""
echo -e "\033[32mEnvironment variables set successfully!\033[0m"
echo -e "\033[33mPlease restart your terminal or run 'source ~/.bashrc' to load the new variables.\033[0m"

echo ""
echo -e "\033[36mTo set DeepSeek API key manually:\033[0m"
echo -e "\033[37mexport DEEPSEEK_API_KEY=\"your-key-here\"\033[0m"
echo -e "\033[36mOr add permanently to ~/.bashrc:\033[0m"
echo -e "\033[37mecho 'export DEEPSEEK_API_KEY=\"your-key-here\"' >> ~/.bashrc\033[0m"

echo ""
echo -e "\033[32mSecurity setup complete!\033[0m"
echo -e "\033[33mJWT_SECRET_KEY set: ${JWT_SECRET:0:8}...\033[0m"

# Check if openssl is available
if ! command -v openssl &> /dev/null; then
    echo ""
    echo -e "\033[31mWarning: openssl not found. Please install it for better security:\033[0m"
    echo -e "\033[37m  Ubuntu/Debian: sudo apt-get install openssl\033[0m"
    echo -e "\033[37m  CentOS/RHEL:   sudo yum install openssl\033[0m"
    echo -e "\033[37m  Arch Linux:    sudo pacman -S openssl\033[0m"
fi

# Make the script remind about permissions
echo ""
echo -e "\033[33mNote: Make sure to set proper file permissions:\033[0m"
echo -e "\033[37mchmod 600 ~/.bashrc\033[0m"
echo -e "\033[37mchmod 700 ~/\033[0m"
