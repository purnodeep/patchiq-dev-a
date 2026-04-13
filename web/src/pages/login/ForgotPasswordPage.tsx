import { useState, forwardRef } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Link } from 'react-router';
import { ArrowLeft, Mail, CheckCircle2 } from 'lucide-react';
import { AuthLayout } from '../../components/auth/AuthLayout';
import { useForgotPassword } from '../../api/hooks/useLogin';

const forgotPasswordSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Please enter a valid email address'),
});

type ForgotPasswordFormValues = z.infer<typeof forgotPasswordSchema>;

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

export function ForgotPasswordPage() {
  const forgotPassword = useForgotPassword();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ForgotPasswordFormValues>({
    resolver: zodResolver(forgotPasswordSchema),
    defaultValues: {
      email: '',
    },
  });

  const onSubmit = (data: ForgotPasswordFormValues) => {
    forgotPassword.mutate(data);
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
        {forgotPassword.isSuccess ? (
          <div
            style={{
              textAlign: 'center',
              display: 'flex',
              flexDirection: 'column',
              gap: '24px',
              alignItems: 'center',
            }}
          >
            <div
              style={{
                display: 'flex',
                height: '56px',
                width: '56px',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '50%',
                background: 'color-mix(in srgb, var(--accent) 10%, transparent)',
                border: '1px solid color-mix(in srgb, var(--accent) 25%, transparent)',
              }}
            >
              <CheckCircle2 style={{ width: '26px', height: '26px', color: 'var(--accent)' }} />
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <p
                style={{
                  fontSize: '16px',
                  fontWeight: 600,
                  color: 'var(--text-emphasis)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Check your email
              </p>
              <p
                style={{
                  fontSize: '13px',
                  color: 'var(--text-secondary)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                If that email exists, we've sent a reset link.
              </p>
            </div>
            <Link
              to="/login"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                fontSize: '13px',
                color: 'var(--accent)',
                textDecoration: 'none',
                fontFamily: 'var(--font-sans)',
              }}
            >
              <ArrowLeft style={{ width: '14px', height: '14px' }} />
              Back to sign in
            </Link>
          </div>
        ) : (
          <>
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
                Reset your password
              </h1>
              <p
                style={{
                  fontSize: '13px',
                  color: 'var(--text-secondary)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Enter your email and we'll send you a reset link
              </p>
            </div>

            <form
              onSubmit={handleSubmit(onSubmit)}
              noValidate
              style={{ display: 'flex', flexDirection: 'column', gap: '18px' }}
            >
              {/* Server error */}
              {forgotPassword.isError && (
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
                  {forgotPassword.error.message}
                </div>
              )}

              {/* Email */}
              <div>
                <label htmlFor="forgot-email" style={labelStyle}>
                  Email
                </label>
                <FocusInput
                  id="forgot-email"
                  type="email"
                  placeholder="you@company.com"
                  autoComplete="email"
                  aria-invalid={!!errors.email}
                  {...register('email')}
                />
                {errors.email && <p style={errorStyle}>{errors.email.message}</p>}
              </div>

              {/* Submit button */}
              <button
                type="submit"
                disabled={forgotPassword.isPending}
                style={{
                  width: '100%',
                  height: '44px',
                  background: forgotPassword.isPending
                    ? 'color-mix(in srgb, var(--accent) 60%, transparent)'
                    : 'var(--accent)',
                  border: 'none',
                  borderRadius: '6px',
                  fontSize: '14px',
                  fontWeight: 600,
                  color: 'var(--text-on-color, #fff)',
                  cursor: forgotPassword.isPending ? 'not-allowed' : 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '8px',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                {forgotPassword.isPending ? (
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
                    Sending...
                  </>
                ) : (
                  <>
                    <Mail style={{ width: '15px', height: '15px' }} />
                    Send reset link
                  </>
                )}
              </button>

              {/* Back to login */}
              <Link
                to="/login"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: '6px',
                  fontSize: '13px',
                  color: 'var(--accent)',
                  textDecoration: 'none',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                <ArrowLeft style={{ width: '14px', height: '14px' }} />
                Back to sign in
              </Link>
            </form>
          </>
        )}
      </div>
    </AuthLayout>
  );
}
