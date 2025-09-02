package email

import (
	"fmt"
	"carryless/internal/models"
)

func (s *Service) generateWelcomeHTML(user *models.User, activationToken string) string {
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
            
            <p><strong>To complete your registration and start using Carryless, please activate your account by clicking the link below:</strong></p>
            
            <p style="text-align: center; margin: 30px 0;">
                <a href="https://carryless.org/activate/%s" class="cta-button">Activate Your Account</a>
            </p>
            
            <p style="font-size: 14px; color: #6c757d;">This activation link will expire in 24 hours.</p>
            
            <p>With Carryless, you can:</p>
            <ul>
                <li>📦 Organize your outdoor gear inventory</li>
                <li>🎒 Plan and optimize your packs for adventures</li>
                <li>⚖️ Track weights and analyze pack distribution</li>
                <li>🌍 Share your pack lists with the community</li>
            </ul>
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
</html>`, user.Username, activationToken, user.Email)
}

func (s *Service) generateWelcomeText(user *models.User, activationToken string) string {
	return fmt.Sprintf(`Welcome %s!

Thank you for joining Carryless, the ultimate outdoor gear catalog and pack planner!

To complete your registration and start using Carryless, please activate your account by visiting:
https://carryless.org/activate/%s

This activation link will expire in 24 hours.

With Carryless, you can:
- Organize your outdoor gear inventory
- Plan and optimize your packs for adventures  
- Track weights and analyze pack distribution
- Share your pack lists with the community

Happy trails!
The Carryless Team

---
This email was sent to %s. If you have any questions, feel free to reach out to us.`, user.Username, activationToken, user.Email)
}

func (s *Service) generateAdminNotificationHTML(admin *models.User, newUser *models.User) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>New User Registration</title>
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
        .notification-title {
            font-size: 24px;
            color: #2d5e3e;
            margin-bottom: 20px;
        }
        .user-info {
            background-color: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
        }
        .user-info h3 {
            color: #2d5e3e;
            margin-top: 0;
        }
        .info-row {
            margin: 10px 0;
        }
        .label {
            font-weight: 600;
            color: #495057;
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
            <div class="logo">Carryless Admin</div>
            <div class="notification-title">New User Registration</div>
        </div>
        
        <div class="content">
            <p>Hello %s,</p>
            <p>A new user has registered on Carryless:</p>
            
            <div class="user-info">
                <h3>User Details</h3>
                <div class="info-row">
                    <span class="label">Username:</span> %s
                </div>
                <div class="info-row">
                    <span class="label">Email:</span> %s
                </div>
                <div class="info-row">
                    <span class="label">Registration Time:</span> %s
                </div>
            </div>
        </div>
        
        <div class="footer">
            <p>Carryless Admin Notification System</p>
        </div>
    </div>
</body>
</html>`, admin.Username, newUser.Username, newUser.Email, newUser.CreatedAt.Format("January 2, 2006 at 3:04 PM"))
}

func (s *Service) generateAdminNotificationText(admin *models.User, newUser *models.User) string {
	return fmt.Sprintf(`New User Registration

Hello %s,

A new user has registered on Carryless:

User Details:
- Username: %s
- Email: %s
- Registration Time: %s

---
Carryless Admin Notification System`, admin.Username, newUser.Username, newUser.Email, newUser.CreatedAt.Format("January 2, 2006 at 3:04 PM"))
}