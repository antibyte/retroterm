#!/bin/bash

# TinyOS Let's Encrypt Setup Script - Linux Version

echo -e "\033[32mTinyOS Let's Encrypt Setup\033[0m"
echo -e "\033[32m============================\033[0m"

# Check if certbot is installed
if ! command -v certbot &> /dev/null; then
    echo ""
    echo -e "\033[31mError: certbot is not installed.\033[0m"
    echo -e "\033[33mPlease install certbot first:\033[0m"
    echo -e "\033[37m  Ubuntu/Debian: sudo apt-get install certbot\033[0m"
    echo -e "\033[37m  CentOS/RHEL:   sudo yum install certbot\033[0m"
    echo -e "\033[37m  Arch Linux:    sudo pacman -S certbot\033[0m"
    echo ""
    echo -e "\033[33mOr install using snap:\033[0m"
    echo -e "\033[37m  sudo snap install --classic certbot\033[0m"
    exit 1
fi

# Prompt for domain
echo ""
read -p "Enter your domain name (e.g., yourdomain.com): " domain
if [[ -z "$domain" ]]; then
    echo -e "\033[31mDomain is required. Exiting.\033[0m"
    exit 1
fi

# Prompt for email
echo ""
read -p "Enter your email address for Let's Encrypt registration: " email
if [[ -z "$email" ]]; then
    echo -e "\033[31mEmail is required. Exiting.\033[0m"
    exit 1
fi

# Ask about HTTPS redirect
echo ""
read -p "Force HTTPS redirect (redirect all HTTP traffic to HTTPS)? (y/N): " redirect_choice
force_redirect="n"
if [[ "$redirect_choice" == "y" || "$redirect_choice" == "Y" ]]; then
    force_redirect="y"
fi

# Ask about ports
echo ""
echo -e "\033[33mPort Configuration:\033[0m"
echo "For Let's Encrypt to work, you need:"
echo "- Port 80 (HTTP) accessible for domain validation"
echo "- Port 443 (HTTPS) accessible for secure connections"
echo ""

read -p "Use standard ports (80/443)? (Y/n): " use_standard_ports
if [[ "$use_standard_ports" == "n" || "$use_standard_ports" == "N" ]]; then
    read -p "Enter HTTP port (default: 8080): " http_port
    read -p "Enter HTTPS port (default: 8443): " https_port
    http_port=${http_port:-8080}
    https_port=${https_port:-8443}
else
    http_port="80"
    https_port="443"
fi

# Create certificate directory
cert_dir="./certs"
mkdir -p "$cert_dir"

echo ""
echo -e "\033[33mCertificate Setup:\033[0m"
echo "Domain: $domain"
echo "Email: $email"
echo "HTTP Port: $http_port"
echo "HTTPS Port: $https_port"
echo "Force HTTPS Redirect: $force_redirect"
echo "Certificate Directory: $cert_dir"

echo ""
read -p "Proceed with certificate generation? (Y/n): " proceed
if [[ "$proceed" == "n" || "$proceed" == "N" ]]; then
    echo "Setup cancelled."
    exit 0
fi

# Stop any running instances of the application
echo ""
echo -e "\033[33mStopping any running TinyOS instances...\033[0m"
pkill -f "./main" 2>/dev/null || true
pkill -f "go run main.go" 2>/dev/null || true

# Generate certificate using certbot standalone
echo ""
echo -e "\033[33mGenerating Let's Encrypt certificate...\033[0m"
echo -e "\033[37mThis may take a few minutes...\033[0m"

certbot_cmd="sudo certbot certonly --standalone"
certbot_cmd="$certbot_cmd --preferred-challenges http"
certbot_cmd="$certbot_cmd --http-01-port $http_port"
certbot_cmd="$certbot_cmd --email $email"
certbot_cmd="$certbot_cmd --agree-tos"
certbot_cmd="$certbot_cmd --no-eff-email"
certbot_cmd="$certbot_cmd -d $domain"

echo "Running: $certbot_cmd"
if eval "$certbot_cmd"; then
    echo ""
    echo -e "\033[32m✓ Certificate generated successfully!\033[0m"
    
    # Copy certificates to application directory
    echo ""
    echo -e "\033[33mCopying certificates to application directory...\033[0m"
    
    sudo cp "/etc/letsencrypt/live/$domain/fullchain.pem" "$cert_dir/"
    sudo cp "/etc/letsencrypt/live/$domain/privkey.pem" "$cert_dir/"
    sudo chown $(whoami):$(whoami) "$cert_dir/fullchain.pem" "$cert_dir/privkey.pem"
    sudo chmod 600 "$cert_dir/privkey.pem"
    sudo chmod 644 "$cert_dir/fullchain.pem"
    
    echo -e "\033[32m✓ Certificates copied to $cert_dir/\033[0m"
    
    # Update settings.cfg
    echo ""
    echo -e "\033[33mUpdating settings.cfg...\033[0m"
    
    # Update TLS settings
    if grep -q "tls_enabled" settings.cfg; then
        sed -i "s/tls_enabled=.*/tls_enabled=true/" settings.cfg
    else
        echo "tls_enabled=true" >> settings.cfg
    fi
    
    if grep -q "tls_cert_file" settings.cfg; then
        sed -i "s|tls_cert_file=.*|tls_cert_file=$cert_dir/fullchain.pem|" settings.cfg
    else
        echo "tls_cert_file=$cert_dir/fullchain.pem" >> settings.cfg
    fi
    
    if grep -q "tls_key_file" settings.cfg; then
        sed -i "s|tls_key_file=.*|tls_key_file=$cert_dir/privkey.pem|" settings.cfg
    else
        echo "tls_key_file=$cert_dir/privkey.pem" >> settings.cfg
    fi
    
    # Update port settings
    if [[ "$http_port" != "80" ]]; then
        if grep -q "port" settings.cfg; then
            sed -i "s/port=.*/port=$http_port/" settings.cfg
        else
            echo "port=$http_port" >> settings.cfg
        fi
    fi
    
    if [[ "$https_port" != "443" ]]; then
        if grep -q "tls_port" settings.cfg; then
            sed -i "s/tls_port=.*/tls_port=$https_port/" settings.cfg
        else
            echo "tls_port=$https_port" >> settings.cfg
        fi
    fi
    
    # Set HTTPS redirect
    if [[ "$force_redirect" == "y" ]]; then
        if grep -q "force_https" settings.cfg; then
            sed -i "s/force_https=.*/force_https=true/" settings.cfg
        else
            echo "force_https=true" >> settings.cfg
        fi
    fi
    
    echo -e "\033[32m✓ settings.cfg updated\033[0m"
    
    # Set up automatic renewal
    echo ""
    echo -e "\033[33mSetting up automatic certificate renewal...\033[0m"
    
    # Create renewal script
    cat > "$cert_dir/renew-cert.sh" << EOF
#!/bin/bash
# TinyOS Certificate Renewal Script

echo "Renewing Let's Encrypt certificate for $domain..."

# Stop TinyOS
pkill -f "./main" 2>/dev/null || true
pkill -f "go run main.go" 2>/dev/null || true

# Renew certificate
sudo certbot renew --quiet

# Copy new certificates
sudo cp "/etc/letsencrypt/live/$domain/fullchain.pem" "$cert_dir/"
sudo cp "/etc/letsencrypt/live/$domain/privkey.pem" "$cert_dir/"
sudo chown \$(whoami):\$(whoami) "$cert_dir/fullchain.pem" "$cert_dir/privkey.pem"
sudo chmod 600 "$cert_dir/privkey.pem"
sudo chmod 644 "$cert_dir/fullchain.pem"

echo "Certificate renewed successfully!"

# Restart TinyOS (optional - uncomment if you want auto-restart)
# cd "$(pwd)"
# nohup ./main &

EOF
    
    chmod +x "$cert_dir/renew-cert.sh"
    
    echo -e "\033[32m✓ Renewal script created at $cert_dir/renew-cert.sh\033[0m"
    
    # Suggest cron job
    echo ""
    echo -e "\033[33mTo set up automatic renewal, add this to your crontab (crontab -e):\033[0m"
    echo -e "\033[37m0 2 * * 0 $cert_dir/renew-cert.sh\033[0m"
    echo -e "\033[37m(This runs every Sunday at 2 AM)\033[0m"
    
    echo ""
    echo -e "\033[32mLet's Encrypt setup complete!\033[0m"
    echo ""
    echo -e "\033[33mNext steps:\033[0m"
    echo "1. Your TinyOS application will now use HTTPS"
    echo "2. Access your site at: https://$domain:$https_port"
    echo "3. HTTP traffic will redirect to HTTPS if enabled"
    echo "4. Certificates will expire in 90 days - set up automatic renewal"
    
else
    echo ""
    echo -e "\033[31mCertificate generation failed!\033[0m"
    echo -e "\033[33mPlease check:\033[0m"
    echo "1. Domain DNS points to this server"
    echo "2. Ports 80 and 443 are open and accessible"
    echo "3. No other web server is running on port $http_port"
    exit 1
fi
