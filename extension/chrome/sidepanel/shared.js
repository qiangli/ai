//

// default selectors for the domain
const selectorMap = {
    'leetcode.com': '[data-track-load="description_content"]',
    'codeforces.com': '.problem-statement',
    'codechef.com': '#problem-statement',
    'geeksforgeeks.org': '.problems_problem_content__Xm_eO',
    'interviewbit.com': '.p-statement',
    'testdome.com': '.copy-protection',
};

function getDefaultSelector(url, val) {
    for (const domain in selectorMap) {
        if (url.includes(domain)) {
            return selectorMap[domain];
        }
    }
    return val;
}

function show(element) {
    element.removeAttribute('hidden');
}

function hide(element) {
    element.setAttribute('hidden', '');
}
