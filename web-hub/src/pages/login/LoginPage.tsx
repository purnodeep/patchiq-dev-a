import { useState, forwardRef } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useNavigate } from 'react-router';
import { Eye, EyeOff } from 'lucide-react';
import { AuthLayout } from '../../components/auth/AuthLayout';
import { useLogin } from '../../api/hooks/useLogin';

const loginSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Please enter a valid email address'),
  password: z.string().min(1, 'Password is required'),
  remember_me: z.boolean(),
});

type LoginFormValues = z.infer<typeof loginSchema>;

const inputStyle: React.CSSProperties = {
  width: '100%',
  height: '42px',
  padding: '0 12px',
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  fontSize: '13px',
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color 0.15s, box-shadow 0.15s',
  boxSizing: 'border-box',
};

const inputFocusStyle: React.CSSProperties = {
  borderColor: 'var(--accent)',
  boxShadow: '0 0 0 2px color-mix(in srgb, var(--accent) 15%, transparent)',
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '12px',
  fontWeight: 500,
  color: 'var(--text-secondary)',
  marginBottom: '6px',
  fontFamily: 'var(--font-sans)',
};

const errorStyle: React.CSSProperties = {
  fontSize: '11px',
  color: 'var(--signal-critical)',
  marginTop: '4px',
  fontFamily: 'var(--font-sans)',
};

const FocusInput = forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  function FocusInput({ style, ...rest }, ref) {
    const [focused, setFocused] = useState(false);
    return (
      <input
        ref={ref}
        style={{ ...inputStyle, ...(focused ? inputFocusStyle : {}), ...style }}
        onFocus={() => setFocused(true)}
        onBlur={(e) => {
          setFocused(false);
          rest.onBlur?.(e);
        }}
        {...rest}
      />
    );
  },
);

export function LoginPage() {
  const navigate = useNavigate();
  const login = useLogin();
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: '',
      password: '',
      remember_me: false,
    },
  });

  const onSubmit = (data: LoginFormValues) => {
    login.mutate(data, {
      onSuccess: () => {
        void navigate('/');
      },
    });
  };

  return (
    <AuthLayout>
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: '12px',
          padding: '32px',
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        <div style={{ textAlign: 'center', marginBottom: '28px' }}>
          <h1
            style={{
              fontSize: '22px',
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              letterSpacing: '-0.02em',
              fontFamily: 'var(--font-sans)',
              marginBottom: '6px',
            }}
          >
            Welcome back
          </h1>
          <p
            style={{
              fontSize: '13px',
              color: 'var(--text-secondary)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            Sign in to the Hub Manager to continue
          </p>
        </div>

        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          style={{ display: 'flex', flexDirection: 'column', gap: '18px' }}
        >
          {/* Server error */}
          {login.isError && (
            <div
              style={{
                borderRadius: '6px',
                background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
                border: '1px solid color-mix(in srgb, var(--signal-critical) 25%, transparent)',
                padding: '10px 12px',
                fontSize: '13px',
                color: 'var(--signal-critical)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              {login.error.message}
            </div>
          )}

          {/* Email */}
          <div>
            <label htmlFor="email" style={labelStyle}>
              Email
            </label>
            <FocusInput
              id="email"
              type="email"
              placeholder="you@company.com"
              autoComplete="email"
              aria-invalid={!!errors.email}
              {...register('email')}
            />
            {errors.email && <p style={errorStyle}>{errors.email.message}</p>}
          </div>

          {/* Password */}
          <div>
            <label htmlFor="password" style={labelStyle}>
              Password
            </label>
            <div style={{ position: 'relative' }}>
              <FocusInput
                id="password"
                type={showPassword ? 'text' : 'password'}
                placeholder="Enter your password"
                autoComplete="current-password"
                aria-invalid={!!errors.password}
                style={{ paddingRight: '42px' }}
                {...register('password')}
              />
              <button
                type="button"
                style={{
                  position: 'absolute',
                  right: '12px',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: 0,
                  display: 'flex',
                  color: 'var(--text-muted)',
                }}
                onClick={() => setShowPassword(!showPassword)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? (
                  <EyeOff style={{ width: '16px', height: '16px' }} />
                ) : (
                  <Eye style={{ width: '16px', height: '16px' }} />
                )}
              </button>
            </div>
            {errors.password && <p style={errorStyle}>{errors.password.message}</p>}
          </div>

          {/* Remember me */}
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <label
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                fontSize: '13px',
                color: 'var(--text-secondary)',
                cursor: 'pointer',
                fontFamily: 'var(--font-sans)',
              }}
            >
              <input
                type="checkbox"
                style={{
                  width: '14px',
                  height: '14px',
                  accentColor: 'var(--accent)',
                  cursor: 'pointer',
                }}
                {...register('remember_me')}
              />
              Remember me
            </label>
          </div>

          {/* Sign in button */}
          <button
            type="submit"
            disabled={login.isPending}
            style={{
              width: '100%',
              height: '44px',
              background: login.isPending
                ? 'color-mix(in srgb, var(--accent) 60%, transparent)'
                : 'var(--accent)',
              border: 'none',
              borderRadius: '6px',
              fontSize: '14px',
              fontWeight: 600,
              color: 'var(--text-on-color, #fff)',
              cursor: login.isPending ? 'not-allowed' : 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              fontFamily: 'var(--font-sans)',
              transition: 'opacity 0.15s',
            }}
          >
            {login.isPending ? (
              <>
                <span
                  style={{
                    width: '16px',
                    height: '16px',
                    borderRadius: '50%',
                    border:
                      '2px solid color-mix(in srgb, var(--text-on-color, #fff) 40%, transparent)',
                    borderTopColor: 'var(--text-on-color, #fff)',
                    animation: 'spin 0.7s linear infinite',
                    display: 'inline-block',
                  }}
                />
                Signing in...
              </>
            ) : (
              'Sign in'
            )}
          </button>
        </form>
      </div>
    </AuthLayout>
  );
}
