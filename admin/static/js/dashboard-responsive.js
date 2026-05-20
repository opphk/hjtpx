let responsiveManager = {
    currentBreakpoint: '',
    breakpoints: {
        xs: 0,
        sm: 576,
        md: 768,
        lg: 992,
        xl: 1200,
        xxl: 1400
    },
    listeners: []
};

function initResponsiveManager() {
    detectBreakpoint();
    setupResizeListener();
    initResponsiveCharts();
    initResponsiveTables();
    initResponsiveCards();
    initResponsiveNavigation();
    initResponsiveWidgets();
    setupOrientationListener();
}

function detectBreakpoint() {
    const width = window.innerWidth;
    let breakpoint = 'xs';

    if (width >= responsiveManager.breakpoints.xxl) {
        breakpoint = 'xxl';
    } else if (width >= responsiveManager.breakpoints.xl) {
        breakpoint = 'xl';
    } else if (width >= responsiveManager.breakpoints.lg) {
        breakpoint = 'lg';
    } else if (width >= responsiveManager.breakpoints.md) {
        breakpoint = 'md';
    } else if (width >= responsiveManager.breakpoints.sm) {
        breakpoint = 'sm';
    }

    if (breakpoint !== responsiveManager.currentBreakpoint) {
        responsiveManager.currentBreakpoint = breakpoint;
        onBreakpointChange(breakpoint);
    }
}

function onBreakpointChange(breakpoint) {
    console.log(`断点变化: ${breakpoint}`);

    responsiveManager.listeners.forEach(listener => {
        if (listener.breakpoints.includes(breakpoint)) {
            listener.callback(breakpoint);
        }
    });

    adjustLayoutForBreakpoint(breakpoint);
    optimizeChartsForBreakpoint(breakpoint);
    adjustCardsForBreakpoint(breakpoint);
}

function setupResizeListener() {
    let resizeTimeout;

    window.addEventListener('resize', () => {
        clearTimeout(resizeTimeout);
        resizeTimeout = setTimeout(() => {
            detectBreakpoint();
        }, 100);
    });
}

function setupOrientationListener() {
    window.addEventListener('orientationchange', () => {
        setTimeout(() => {
            detectBreakpoint();
            window.dispatchEvent(new Event('resize'));
        }, 100);
    });
}

function onResize(callback, breakpoints = ['xs', 'sm', 'md', 'lg', 'xl', 'xxl']) {
    responsiveManager.listeners.push({ callback, breakpoints });
}

function adjustLayoutForBreakpoint(breakpoint) {
    const contentWrapper = document.querySelector('.content-wrapper');
    if (!contentWrapper) return;

    switch (breakpoint) {
        case 'xs':
        case 'sm':
            contentWrapper.style.padding = '0.5rem';
            break;
        case 'md':
            contentWrapper.style.padding = '1rem';
            break;
        case 'lg':
        case 'xl':
        case 'xxl':
            contentWrapper.style.padding = '';
            break;
    }
}

function initResponsiveCharts() {
    const charts = document.querySelectorAll('canvas, [id*="Chart"]');

    charts.forEach(chart => {
        const parent = chart.closest('.card-body, .chart-container');
        if (parent) {
            makeChartResponsive(chart, parent);
        }
    });
}

function makeChartResponsive(chart, container) {
    const resizeObserver = new ResizeObserver(entries => {
        for (const entry of entries) {
            const { width, height } = entry.contentRect;
            chart.style.width = width + 'px';
            chart.style.height = height + 'px';

            if (chart.chart) {
                chart.chart.resize();
            }
        }
    });

    resizeObserver.observe(container);
}

function optimizeChartsForBreakpoint(breakpoint) {
    const chartConfigs = {
        xs: { maxPoints: 10, hideLegend: true, reduceAnimations: true },
        sm: { maxPoints: 15, hideLegend: false, reduceAnimations: true },
        md: { maxPoints: 20, hideLegend: false, reduceAnimations: false },
        lg: { maxPoints: 30, hideLegend: false, reduceAnimations: false },
        xl: { maxPoints: 60, hideLegend: false, reduceAnimations: false },
        xxl: { maxPoints: 100, hideLegend: false, reduceAnimations: false }
    };

    const config = chartConfigs[breakpoint];

    if (typeof Chart !== 'undefined') {
        Chart.defaults.animation = config.reduceAnimations ? false : { duration: 1000 };
    }

    Object.values(window.dashboardCharts?.instances || {}).forEach(chart => {
        if (chart && chart.options?.plugins?.legend) {
            chart.options.plugins.legend.display = !config.hideLegend;
            chart.update('none');
        }
    });
}

function initResponsiveTables() {
    const tables = document.querySelectorAll('.table-responsive, .table');

    tables.forEach(table => {
        if (!table.closest('.table-responsive')) {
            const wrapper = document.createElement('div');
            wrapper.className = 'table-responsive';
            table.parentNode.insertBefore(wrapper, table);
            wrapper.appendChild(table);
        }
    });

    const tableHeaders = document.querySelectorAll('th[scope="col"]');
    tableHeaders.forEach(header => {
        header.setAttribute('tabindex', '0');
        header.setAttribute('role', 'columnheader');
    });
}

function initResponsiveCards() {
    const cards = document.querySelectorAll('.card, .small-box, .info-box');

    cards.forEach(card => {
        card.classList.add('responsive-card');

        const resizeObserver = new ResizeObserver(entries => {
            for (const entry of entries) {
                const { width } = entry.contentRect;
                adjustCardForWidth(card, width);
            }
        });

        resizeObserver.observe(card);
    });
}

function adjustCardForWidth(card, width) {
    const title = card.querySelector('.card-title, h3, .info-box-text');
    const description = card.querySelector('.card-body p, small');

    if (width < 300) {
        card.classList.add('compact-mode');
        if (title) {
            title.style.fontSize = '';
        }
    } else if (width < 400) {
        card.classList.remove('compact-mode');
        if (title) {
            title.style.fontSize = '1.2rem';
        }
    } else {
        card.classList.remove('compact-mode');
        if (title) {
            title.style.fontSize = '';
        }
    }
}

function adjustCardsForBreakpoint(breakpoint) {
    const cards = document.querySelectorAll('.card, .small-box');

    cards.forEach(card => {
        if (breakpoint === 'xs' || breakpoint === 'sm') {
            card.classList.add('mobile-layout');
            card.classList.remove('desktop-layout');
        } else {
            card.classList.remove('mobile-layout');
            card.classList.add('desktop-layout');
        }
    });
}

function initResponsiveNavigation() {
    const sidebar = document.querySelector('.main-sidebar');
    const pushMenuBtn = document.querySelector('[data-widget="pushmenu"]');

    if (!sidebar || !pushMenuBtn) return;

    onResize((bp) => {
        if (bp === 'xs' || bp === 'sm') {
            sidebar.classList.add('collapsed');
            sidebar.classList.remove('expanded');
        } else {
            sidebar.classList.remove('collapsed');
            sidebar.classList.add('expanded');
        }
    }, ['xs', 'sm', 'md', 'lg', 'xl', 'xxl']);

    pushMenuBtn.addEventListener('click', () => {
        sidebar.classList.toggle('sidebar-open');
    });
}

function initResponsiveWidgets() {
    const widgetContainers = document.querySelectorAll('[class*="col-"]');

    widgetContainers.forEach(container => {
        if (container.querySelector('.card, .small-box, .info-box')) {
            container.classList.add('widget-container');
        }
    });
}

function getCurrentBreakpoint() {
    return responsiveManager.currentBreakpoint;
}

function isMobile() {
    return ['xs', 'sm'].includes(responsiveManager.currentBreakpoint);
}

function isTablet() {
    return responsiveManager.currentBreakpoint === 'md';
}

function isDesktop() {
    return ['lg', 'xl', 'xxl'].includes(responsiveManager.currentBreakpoint);
}

function getDeviceType() {
    if (isMobile()) return 'mobile';
    if (isTablet()) return 'tablet';
    return 'desktop';
}

function optimizeForDevice() {
    const device = getDeviceType();

    document.body.classList.remove('device-mobile', 'device-tablet', 'device-desktop');
    document.body.classList.add(`device-${device}`);

    switch (device) {
        case 'mobile':
            optimizeForMobile();
            break;
        case 'tablet':
            optimizeForTablet();
            break;
        case 'desktop':
            optimizeForDesktop();
            break;
    }
}

function optimizeForMobile() {
    const largeCharts = document.querySelectorAll('.card > .card-body > canvas');
    largeCharts.forEach(chart => {
        chart.style.maxHeight = '200px';
    });

    const longTables = document.querySelectorAll('.table');
    longTables.forEach(table => {
        table.style.fontSize = '0.85rem';
    });

    hideNonEssentialElements();
}

function optimizeForTablet() {
    const largeCharts = document.querySelectorAll('.card > .card-body > canvas');
    largeCharts.forEach(chart => {
        chart.style.maxHeight = '250px';
    });

    adjustGridColumns(2);
}

function optimizeForDesktop() {
    const charts = document.querySelectorAll('.card > .card-body > canvas');
    charts.forEach(chart => {
        chart.style.maxHeight = '';
    });

    adjustGridColumns(4);
}

function hideNonEssentialElements() {
    const nonEssential = document.querySelectorAll('.hide-mobile, [data-hide-mobile]');
    nonEssential.forEach(el => {
        el.style.display = 'none';
    });
}

function adjustGridColumns(columns) {
    const grids = document.querySelectorAll('[class*="row"]');
    grids.forEach(grid => {
        const currentCols = grid.querySelectorAll('[class*="col-"]');
        if (currentCols.length > columns) {
            grid.dataset.columns = columns;
        }
    });
}

function scrollToElement(selector, behavior = 'smooth') {
    const element = document.querySelector(selector);
    if (element) {
        element.scrollIntoView({ behavior, block: 'start' });
    }
}

function setupTouchOptimizations() {
    if ('ontouchstart' in window) {
        document.body.classList.add('touch-device');

        const buttons = document.querySelectorAll('.btn');
        buttons.forEach(btn => {
            btn.classList.add('touch-optimized');
        });
    }
}

function getOrientation() {
    return window.innerHeight > window.innerWidth ? 'portrait' : 'landscape';
}

function handleOrientationChange() {
    const orientation = getOrientation();
    document.body.classList.remove('orientation-portrait', 'orientation-landscape');
    document.body.classList.add(`orientation-${orientation}`);

    if (orientation === 'landscape' && isMobile()) {
        optimizeForLandscape();
    }
}

function optimizeForLandscape() {
    const cards = document.querySelectorAll('.card');
    cards.forEach(card => {
        card.classList.add('landscape-mode');
    });
}

function lazyLoadContent() {
    const lazyElements = document.querySelectorAll('[data-lazy-load]');

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const element = entry.target;
                const source = element.dataset.lazyLoad;

                if (source === 'chart') {
                    initChart(element);
                } else if (source === 'image') {
                    loadImage(element);
                }

                observer.unobserve(element);
            }
        });
    });

    lazyElements.forEach(el => observer.observe(el));
}

function initChart(element) {
    const chartId = element.id;
    const chartType = element.dataset.chartType;

    if (chartType === 'line' && window.dashboardCharts?.instances?.trend) {
        window.dashboardCharts.instances.trend.update();
    }
}

function loadImage(element) {
    const src = element.dataset.src;
    if (src) {
        element.src = src;
    }
}

document.addEventListener('DOMContentLoaded', function() {
    if (document.getElementById('dashboardContent')) {
        initResponsiveManager();
        optimizeForDevice();
        setupTouchOptimizations();
        lazyLoadContent();

        window.addEventListener('orientationchange', handleOrientationChange);
    }
});

window.responsiveManager = responsiveManager;
window.getCurrentBreakpoint = getCurrentBreakpoint;
window.isMobile = isMobile;
window.isTablet = isTablet;
window.isDesktop = isDesktop;
window.getDeviceType = getDeviceType;
window.scrollToElement = scrollToElement;
