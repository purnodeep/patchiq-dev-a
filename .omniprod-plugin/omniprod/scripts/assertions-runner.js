// OmniProd Assertions Runner — injected into page via evaluate_script
// Usage: evaluate_script with this file's content + assertion definitions
// Returns: { results: [...], summary: { total, passed, failed } }

(function runAssertions(assertionDefs) {
  'use strict';

  const results = [];

  // ---- Built-in assertion categories ----

  // 1. Console/error indicators in DOM
  function checkConsoleErrors() {
    const errorBoundaries = document.querySelectorAll(
      '[data-testid*="error"], .error-boundary, [role="alert"][class*="error"], [class*="ErrorBoundary"]',
    );
    return {
      id: 'BUILTIN-CONSOLE-001',
      category: 'console',
      description: 'No React error boundaries or DOM error indicators visible',
      passed: errorBoundaries.length === 0,
      actual:
        errorBoundaries.length === 0
          ? 'No error indicators'
          : errorBoundaries.length + ' error indicator(s) found',
      expected: 'No error indicators',
      severity: 'critical',
      elements: Array.from(errorBoundaries)
        .map(function (el) {
          return el.className || el.tagName;
        })
        .slice(0, 5),
    };
  }

  // 2. Accessibility basics
  function checkA11yBasics() {
    var checks = [];

    // 2a. All images have alt text
    var images = document.querySelectorAll('img:not([alt])');
    checks.push({
      id: 'BUILTIN-A11Y-001',
      category: 'accessibility',
      description: 'All images have alt attributes',
      passed: images.length === 0,
      actual: images.length === 0 ? 'All images have alt' : images.length + ' image(s) missing alt',
      expected: 'All images have alt attributes',
      severity: 'major',
    });

    // 2b. All form inputs have associated labels
    var inputs = document.querySelectorAll(
      'input:not([type="hidden"]):not([type="submit"]):not([type="button"]), select, textarea',
    );
    var unlabeled = 0;
    inputs.forEach(function (input) {
      var hasLabel = input.id && document.querySelector('label[for="' + input.id + '"]');
      var hasAriaLabel = input.getAttribute('aria-label') || input.getAttribute('aria-labelledby');
      var wrappedInLabel = input.closest('label');
      if (!hasLabel && !hasAriaLabel && !wrappedInLabel) unlabeled++;
    });
    checks.push({
      id: 'BUILTIN-A11Y-002',
      category: 'accessibility',
      description: 'All form inputs have associated labels',
      passed: unlabeled === 0,
      actual: unlabeled === 0 ? 'All inputs labeled' : unlabeled + ' input(s) without labels',
      expected: 'All inputs have labels',
      severity: 'major',
    });

    // 2c. Page has heading structure
    var h1s = document.querySelectorAll('h1');
    checks.push({
      id: 'BUILTIN-A11Y-003',
      category: 'accessibility',
      description: 'Page has exactly one h1 heading',
      passed: h1s.length === 1,
      actual: h1s.length + ' h1 heading(s)',
      expected: 'Exactly 1 h1',
      severity: 'minor',
    });

    // 2d. All buttons have accessible names
    var buttons = document.querySelectorAll('button, [role="button"]');
    var unnamedButtons = 0;
    buttons.forEach(function (btn) {
      var text = (btn.textContent || '').trim();
      var ariaLabel =
        btn.getAttribute('aria-label') ||
        btn.getAttribute('aria-labelledby') ||
        btn.getAttribute('title');
      if (!text && !ariaLabel) unnamedButtons++;
    });
    checks.push({
      id: 'BUILTIN-A11Y-004',
      category: 'accessibility',
      description: 'All buttons have accessible names',
      passed: unnamedButtons === 0,
      actual:
        unnamedButtons === 0 ? 'All buttons named' : unnamedButtons + ' button(s) without names',
      expected: 'All buttons have text or aria-label',
      severity: 'major',
    });

    // 2e. Main landmark exists
    var main = document.querySelector('main, [role="main"]');
    checks.push({
      id: 'BUILTIN-A11Y-005',
      category: 'accessibility',
      description: 'Page has a main landmark',
      passed: !!main,
      actual: main ? 'Main landmark found' : 'No main landmark',
      expected: 'Page has <main> or role="main"',
      severity: 'minor',
    });

    return checks;
  }

  // 3. Data integrity basics
  function checkDataIntegrity() {
    var checks = [];

    // 3a. No visible "undefined", "null", "NaN" in text content
    var bodyText = document.body.innerText || '';
    var badPatterns = [
      { pattern: /\bundefined\b/g, name: 'undefined' },
      { pattern: /\bnull\b/gi, name: 'null' },
      { pattern: /\bNaN\b/g, name: 'NaN' },
      { pattern: /\[object Object\]/g, name: '[object Object]' },
    ];
    var found = [];
    badPatterns.forEach(function (bp) {
      var matches = bodyText.match(bp.pattern);
      if (matches) found.push(bp.name + ' (' + matches.length + 'x)');
    });
    checks.push({
      id: 'BUILTIN-DI-001',
      category: 'data-integrity',
      description: 'No raw undefined/null/NaN/[object Object] visible in page text',
      passed: found.length === 0,
      actual: found.length === 0 ? 'Clean text content' : 'Found: ' + found.join(', '),
      expected: 'No raw JS values in visible text',
      severity: 'critical',
    });

    // 3b. Tables have data or empty state
    var tables = document.querySelectorAll('table');
    tables.forEach(function (table, i) {
      var rows = table.querySelectorAll('tbody tr');
      var parent = table.closest('[class]');
      var emptyState =
        parent &&
        (parent.querySelector('[class*="empty"]') ||
          parent.querySelector('[class*="Empty"]') ||
          parent.querySelector('[data-testid*="empty"]'));
      checks.push({
        id: 'BUILTIN-DI-002-' + i,
        category: 'data-integrity',
        description: 'Table ' + (i + 1) + ' has data rows or explicit empty state',
        passed: rows.length > 0 || !!emptyState,
        actual:
          rows.length > 0
            ? rows.length + ' rows'
            : emptyState
              ? 'Empty state shown'
              : 'No rows AND no empty state',
        expected: 'Data rows or empty state component',
        severity: rows.length === 0 && !emptyState ? 'major' : 'minor',
      });
    });

    return checks;
  }

  // 4. Performance basics
  function checkPerformanceBasics() {
    var checks = [];
    var allElements = document.querySelectorAll('*').length;
    checks.push({
      id: 'BUILTIN-PERF-001',
      category: 'performance',
      description: 'DOM element count is reasonable (<3000)',
      passed: allElements < 3000,
      actual: allElements + ' elements',
      expected: '<3000 DOM elements',
      severity: allElements > 5000 ? 'major' : 'minor',
    });
    return checks;
  }

  // 5. Custom assertions from assertion-defs.json
  function runCustomAssertions(defs) {
    if (!defs || !Array.isArray(defs)) return [];
    var customResults = [];

    for (var idx = 0; idx < defs.length; idx++) {
      var def = defs[idx];
      try {
        var fn = new Function('document', 'window', def.script);
        var result = fn(document, window);
        customResults.push({
          id: def.id,
          category: def.category || 'custom',
          description: def.description,
          passed: !!result.passed,
          actual: String(result.actual || ''),
          expected: String(result.expected || ''),
          severity: def.severity || 'major',
        });
      } catch (err) {
        customResults.push({
          id: def.id,
          category: def.category || 'custom',
          description: def.description,
          passed: false,
          actual: 'Error: ' + err.message,
          expected: 'Assertion should run without errors',
          severity: def.severity || 'major',
        });
      }
    }
    return customResults;
  }

  // ---- Run all assertions ----
  results.push(checkConsoleErrors());
  results.push.apply(results, checkA11yBasics());
  results.push.apply(results, checkDataIntegrity());
  results.push.apply(results, checkPerformanceBasics());

  if (assertionDefs && assertionDefs.length > 0) {
    results.push.apply(results, runCustomAssertions(assertionDefs));
  }

  var passed = results.filter(function (r) {
    return r.passed;
  }).length;
  var failed = results.filter(function (r) {
    return !r.passed;
  }).length;

  return JSON.stringify({
    results: results,
    summary: {
      total: results.length,
      passed: passed,
      failed: failed,
      timestamp: new Date().toISOString(),
    },
  });
})(typeof __ASSERTION_DEFS__ !== 'undefined' ? __ASSERTION_DEFS__ : []);
