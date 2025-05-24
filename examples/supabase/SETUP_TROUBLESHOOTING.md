# Supabase Database Setup Guide

## Issue: Database Connection Problems

If you're experiencing "Tenant or user not found" or IPv6 connection issues, follow these steps:

## Step 1: Set Up Database Schema (REQUIRED)

### Option A: Using Supabase Dashboard (Recommended)

1. **Go to your Supabase project dashboard** at https://app.supabase.com
2. **Navigate to SQL Editor** in the left sidebar  
3. **Click "New Query"**
4. **Copy and paste the entire `init.sql` file content** into the editor
5. **Click "Run"** to execute the script
6. **Verify success** - you should see tables created in the `userprefs` schema

### Option B: Using a PostgreSQL Client

If you have `psql` installed:

```bash
# Get your connection string from Supabase dashboard
# Settings → Database → Connection string (Direct connection)
PGPASSWORD="your-password" psql "postgresql://postgres:[password]@db.[project-ref].supabase.co:5432/postgres" -f init.sql
```

## Step 2: Fix IPv6 Connection Issues

If you're getting IPv6 connection timeouts, try these solutions:

### Solution 1: Use Connection Pooler

Update your `.env` file:

```env
# Use the pooler instead of direct connection
SUPABASE_DB_URL=postgresql://postgres:[password]@aws-0-us-east-1.pooler.supabase.com:6543/postgres?sslmode=require
```

### Solution 2: Force IPv4 with System Settings

**On macOS:**

Add to your `~/.zshrc` or `~/.bash_profile`:

```bash
export GODEBUG=netdns=go+1
```

Then restart your terminal.

### Solution 3: Use Direct IP Address

1. **Find the IPv4 address:**
   ```bash
   nslookup aws-0-us-east-1.pooler.supabase.com
   ```

2. **Update your .env file with the IP:**
   ```env
   SUPABASE_DB_URL=postgresql://postgres:[password]@[IPv4-ADDRESS]:6543/postgres?sslmode=require
   ```

## Step 3: Verify Your Connection String Format

Make sure your `.env` file uses the correct format. Check your Supabase dashboard:

1. **Go to Settings → Database**
2. **Copy the "Connection string" under "Direct connection"**
3. **Replace `[YOUR-PASSWORD]` with your actual database password**

Example correct format:
```env
SUPABASE_DB_URL=postgresql://postgres:your-password@db.your-project-ref.supabase.co:5432/postgres?sslmode=require
```

## Step 4: Test the Connection

After setting up the schema and fixing connection issues:

```bash
make run
```

## Troubleshooting

### "Tenant or user not found"
- This usually means the database schema hasn't been set up yet
- Follow Step 1 to create the required tables

### IPv6 Connection Timeouts
- Your network might not support IPv6 properly
- Follow Step 2 to force IPv4 or use the pooler

### Authentication Errors
- Double-check your database password in Supabase dashboard
- Make sure you're using the correct project reference

### Still Having Issues?

1. **Check Supabase status**: https://status.supabase.com
2. **Try the local Docker option**: `make run-local`
3. **Contact support**: The Supabase community is very helpful!

## Local Development Alternative

If cloud connection continues to be problematic:

```bash
# Use local Docker environment instead
make run-local
```

This starts local PostgreSQL and Redis containers for development.
