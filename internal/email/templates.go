package email

import (
	"fmt"
	"carryless/internal/models"
)

func (s *Service) generateWelcomeHTML(user *models.User) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to Carryless</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f8f9fa;
        }
        .container {
            background-color: white;
            padding: 40px;
            border-radius: 12px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            font-size: 28px;
            font-weight: bold;
            color: #2d5e3e;
            margin-bottom: 10px;
        }
        .welcome-message {
            font-size: 24px;
            color: #2d5e3e;
            margin-bottom: 20px;
        }
        .content {
            font-size: 16px;
            margin-bottom: 30px;
        }
        .cta-button {
            display: inline-block;
            background-color: #2d5e3e;
            color: white;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 500;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #e9ecef;
            font-size: 14px;
            color: #6c757d;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">Carryless</div>
            <div class="welcome-message">Welcome %s!</div>
        </div>
        
        <div class="content">
            <p>Thank you for joining Carryless, the ultimate outdoor gear catalog and pack planner!</p>
            
            <p>With Carryless, you can:</p>
            <ul>
                <li>📦 Organize your outdoor gear inventory</li>
                <li>🎒 Plan and optimize your packs for adventures</li>
                <li>⚖️ Track weights and analyze pack distribution</li>
                <li>🌍 Share your pack lists with the community</li>
            </ul>
            
            <p>Ready to start planning your next adventure?</p>
            
            <p style="text-align: center; margin: 30px 0;">
                <a href="https://carryless.org/dashboard" class="cta-button">Start Organizing Your Gear</a>
            </p>
        </div>
        
        <div class="footer">
            <p>Happy trails!</p>
            <p>The Carryless Team</p>
            <p style="margin-top: 20px; font-size: 12px;">
                This email was sent to %s. If you have any questions, feel free to reach out to us.
            </p>
        </div>
    </div>
</body>
</html>`, user.Username, user.Email)
}

func (s *Service) generateWelcomeText(user *models.User) string {
	return fmt.Sprintf(`Welcome %s!

Thank you for joining Carryless, the ultimate outdoor gear catalog and pack planner!

With Carryless, you can:
- Organize your outdoor gear inventory
- Plan and optimize your packs for adventures  
- Track weights and analyze pack distribution
- Share your pack lists with the community

Ready to start planning your next adventure? Visit your dashboard at:
https://carryless.org/dashboard

Happy trails!
The Carryless Team

---
This email was sent to %s. If you have any questions, feel free to reach out to us.`, user.Username, user.Email)
}