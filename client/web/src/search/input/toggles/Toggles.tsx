import React, { useCallback } from 'react'
import * as H from 'history'
import RegexIcon from 'mdi-react/RegexIcon'
import classNames from 'classnames'
import FormatLetterCaseIcon from 'mdi-react/FormatLetterCaseIcon'
import { PatternTypeProps, CaseSensitivityProps, CopyQueryButtonProps } from '../..'
import { SettingsCascadeProps } from '../../../../../shared/src/settings/settings'
import { submitSearch } from '../../helpers'
import { QueryInputToggle } from './QueryInputToggle'
import { isErrorLike } from '../../../../../shared/src/util/errors'
import CodeBracketsIcon from 'mdi-react/CodeBracketsIcon'
import { CopyQueryButton } from './CopyQueryButton'
import { VersionContextProps } from '../../../../../shared/src/search/util'
import { SearchPatternType } from '../../../graphql-operations'
import { findFilter, FilterKind } from '../../../../../shared/src/search/query/validate'

export interface TogglesProps
    extends PatternTypeProps,
        CaseSensitivityProps,
        SettingsCascadeProps,
        CopyQueryButtonProps,
        VersionContextProps {
    navbarSearchQuery: string
    history: H.History
    location: H.Location
    hasGlobalQueryBehavior?: boolean
    className?: string
}

/**
 * The toggles displayed in the query input.
 */
export const Toggles: React.FunctionComponent<TogglesProps> = (props: TogglesProps) => {
    const {
        history,
        navbarSearchQuery,
        versionContext,
        hasGlobalQueryBehavior,
        patternType,
        setPatternType,
        caseSensitive,
        setCaseSensitivity,
        settingsCascade,
        className,
        copyQueryButton,
    } = props

    const structuralSearchDisabled = window.context?.experimentalFeatures?.structuralSearch === 'disabled'

    const submitOnToggle = useCallback(
        (args: { newPatternType: SearchPatternType } | { newCaseSensitivity: boolean }): void => {
            // Only submit search on toggle when the query input has global behavior (i.e. it's on the main search page
            // or global navbar). Non-global inputs don't have the canonical query and need more context, making
            // submit on-toggle undesirable. Also, only submit on toggle only when the query is non-empty.
            const searchQueryNotEmpty = navbarSearchQuery !== ''
            const shouldSubmitSearch = hasGlobalQueryBehavior && searchQueryNotEmpty
            if (shouldSubmitSearch) {
                const activation = undefined
                const source = 'filter'
                const newPatternType = 'newPatternType' in args ? args.newPatternType : patternType
                const newCaseSensitive = 'newCaseSensitivity' in args ? args.newCaseSensitivity : caseSensitive
                submitSearch({
                    history,
                    query: navbarSearchQuery,
                    source,
                    patternType: newPatternType,
                    caseSensitive: newCaseSensitive,
                    versionContext,
                    activation,
                })
            }
        },
        [caseSensitive, hasGlobalQueryBehavior, history, navbarSearchQuery, patternType, versionContext]
    )

    const toggleCaseSensitivity = useCallback((): void => {
        const newCaseSensitivity = !caseSensitive
        setCaseSensitivity(newCaseSensitivity)
        submitOnToggle({ newCaseSensitivity })
    }, [caseSensitive, setCaseSensitivity, submitOnToggle])

    const toggleRegexp = useCallback((): void => {
        const newPatternType =
            patternType !== SearchPatternType.regexp ? SearchPatternType.regexp : SearchPatternType.literal

        setPatternType(newPatternType)
        submitOnToggle({ newPatternType })
    }, [patternType, setPatternType, submitOnToggle])

    const toggleStructuralSearch = useCallback((): void => {
        const cascadePatternTypeValue =
            settingsCascade.final &&
            !isErrorLike(settingsCascade.final) &&
            (settingsCascade.final['search.defaultPatternType'] as SearchPatternType)

        const defaultPatternType = cascadePatternTypeValue || SearchPatternType.literal

        const newPatternType: SearchPatternType =
            patternType !== SearchPatternType.structural ? SearchPatternType.structural : defaultPatternType

        setPatternType(newPatternType)
        submitOnToggle({ newPatternType })
    }, [patternType, setPatternType, settingsCascade.final, submitOnToggle])

    const fullQuery = [navbarSearchQuery, `patternType:${patternType}`, caseSensitive ? 'case:yes' : '']
        .filter(queryPart => !!queryPart)
        .join(' ')

    return (
        <div className={classNames('toggle-container', className)}>
            {copyQueryButton && (
                <CopyQueryButton
                    fullQuery={fullQuery}
                    className="toggle-container__toggle toggle-container__copy-query-button"
                />
            )}
            <QueryInputToggle
                {...props}
                title="Case sensitivity"
                isActive={caseSensitive}
                onToggle={toggleCaseSensitivity}
                icon={FormatLetterCaseIcon}
                className="test-case-sensitivity-toggle"
                activeClassName="test-case-sensitivity-toggle--active"
                disableOn={[
                    {
                        condition: findFilter(navbarSearchQuery, 'case', FilterKind.Subexpression) !== undefined,
                        reason: 'Query already contains one or more case subexpressions',
                    },
                    {
                        condition: findFilter(navbarSearchQuery, 'patterntype', FilterKind.Subexpression) !== undefined,
                        reason:
                            'Query contains one or more patterntype subexpressions, cannot apply global case-sensitivity',
                    },
                    {
                        condition: patternType === SearchPatternType.structural,
                        reason: 'Structural search is always case sensitive',
                    },
                ]}
            />
            <QueryInputToggle
                {...props}
                title="Regular expression"
                isActive={patternType === SearchPatternType.regexp}
                onToggle={toggleRegexp}
                icon={RegexIcon}
                className="test-regexp-toggle"
                activeClassName="test-regexp-toggle--active"
                disableOn={[
                    {
                        condition: findFilter(navbarSearchQuery, 'patterntype', FilterKind.Subexpression) !== undefined,
                        reason: 'Query already contains one or more patterntype subexpressions',
                    },
                ]}
            />
            {!structuralSearchDisabled && (
                <QueryInputToggle
                    {...props}
                    title="Structural search"
                    className="test-structural-search-toggle"
                    activeClassName="test-structural-search-toggle--active"
                    isActive={patternType === SearchPatternType.structural}
                    onToggle={toggleStructuralSearch}
                    icon={CodeBracketsIcon}
                    disableOn={[
                        {
                            condition:
                                findFilter(navbarSearchQuery, 'patterntype', FilterKind.Subexpression) !== undefined,
                            reason: 'Query already contains one or more patterntype subexpressions',
                        },
                    ]}
                />
            )}
        </div>
    )
}
