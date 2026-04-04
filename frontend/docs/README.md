Overview

This project is a responsive captive portal interface designed for Wi-Fi authentication and onboarding. It provides a clean, modern UI for users to connect to a network using a phone number, guest access, or voucher-based authentication.
The interface is built using HTML, TailwindCSS, and vanilla JavaScript, with a focus on usability, accessibility, and visual polish.


Features
1. User Authentication Options
Phone Number Login
Users enter their phone number and receive a verification flow (external integration expected).
Includes validation and normalization of MSISDN format.
Guest Login
Generates a temporary guest identity.
Stores session data locally and redirects to packages page.
Voucher Login
Allows access via prepaid vouchers.
Uses session-based authentication state.

3. UI/UX Design
Fully responsive layout (mobile-first)
TailwindCSS-based styling
Dark mode support
Glassmorphism and animated background effects
Accessible form inputs and feedback states

4. Session Management

Uses sessionStorage to persist user session data:

gh_auth_type → authentication method (phone, guest, voucher)
gh_user → user identifier
gh_msisdn → normalized phone number (for phone login)
4. Validation Logic
Phone number normalization:
Removes non-digit characters
Handles leading zero removal
Validates length (7–15 digits)
Formats to international standard 
Terms of Service acceptance required

File Structure
project/
│
├── index.html          # Main captive portal page
├── packages.html       # Redirect target after login (not included here)
├── images/
│   └── logo.jpeg       # Company logo

Technologies Used
HTML5
TailwindCSS (CDN)
JavaScript (Vanilla)
Google Fonts (Inter)
Material Icons

How It Works
User lands on the captive portal page.
Chooses one of the authentication methods:
Phone number
Guest login
Voucher login
JavaScript processes input and stores session data.
User is redirected to packages.html for plan selection or further steps.
Key Functions
normalizeMsisdn(countryCode, rawPhone)

Formats and validates phone numbers into international format.

randomGuest()
Generates a temporary guest username.
goPackages()
Redirects user to the next step (packages.html).

You can easily modify:
Branding
Logo: images/logo.jpeg
Colors: Tailwind config section in <script id="tailwind-config">
Authentication Logic
Replace sessionStorage with API calls
Integrate SMS/OTP services (e.g., Africa’s Talking, Twilio)

Navigation
window.location.href = "packages.html";
Limitations
No backend integration (pure frontend prototype)
No real authentication (session-based simulation only)
OTP flow not implemented
Voucher validation not connected to a database
Recommended Improvements
Integrate backend authentication API
Add OTP verification flow
Implement real voucher validation
Add analytics (user sessions, conversions)
Secure session handling (JWT or server-side sessions)

License
© 2026 GORNHOM Innovation Ltd. All rights reserved.
Support
For issues or customization requests:
Phone: 0800-GORNHOM
Internal support integration can be added to the "Contact Support" link
